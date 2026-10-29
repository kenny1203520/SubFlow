import { spawn } from 'node:child_process'
import { mkdtemp, rm, writeFile } from 'node:fs/promises'
import net from 'node:net'
import { tmpdir } from 'node:os'
import { join, resolve } from 'node:path'

const frontendDir = resolve(import.meta.dirname, '..')
const repoDir = resolve(frontendDir, '..')
const backendDir = join(repoDir, 'backend')
const dataDir = await mkdtemp(join(tmpdir(), 'subflow-playwright-'))
const envFile = join(dataDir, '.env')
await writeFile(envFile, 'SUBFLOW_SETTINGS_ENCRYPTION_KEY="subflow-playwright-isolated-test-key"\n', { mode: 0o600 })
const backendPort = await new Promise((resolvePort, rejectPort) => {
  const server = net.createServer()
  server.once('error', rejectPort)
  server.listen(0, '127.0.0.1', () => {
    const port = server.address().port
    server.close(() => resolvePort(port))
  })
})
const backendURL = `http://127.0.0.1:${backendPort}`
const children = []
let backendOutput = ''
let stopping = false

function safeBackendDiagnostics() {
  return backendOutput
    .replace(/([?&](?:token|key|password|secret)=)[^\s&]+/gi, '$1[redacted]')
    .replace(/\bBearer\s+\S+/gi, 'Bearer [redacted]')
    .replace(/\b(password|secret|token|key)(\s*[:=]\s*)\S+/gi, '$1$2[redacted]')
    .split(/\r?\n/)
    .filter(line => /\b(fatal|panic|error|failed|cannot)\b/i.test(line))
    .slice(-8)
    .join('\n')
}

function start(command, args, options) {
  const child = spawn(command, args, { stdio: ['ignore', 'pipe', 'pipe'], ...options })
  children.push(child)
  return child
}

function stop() {
  if (stopping) return
  stopping = true
  for (const child of children) {
    if (process.platform === 'win32' && child.pid) spawn('taskkill', ['/pid', String(child.pid), '/t', '/f'], { stdio: 'ignore', windowsHide: true })
    else child.kill('SIGTERM')
  }
  setTimeout(async () => {
    for (const child of children) if (child.exitCode === null) child.kill('SIGKILL')
    await rm(dataDir, { recursive: true, force: true })
    process.exit(0)
  }, 2500).unref()
}
process.on('SIGINT', stop)
process.on('SIGTERM', stop)
process.on('exit', () => { for (const child of children) child.kill('SIGKILL') })

const backendEnvironment = { ...process.env, SUBFLOW_ENV: 'development', SUBFLOW_HTTP_PORT: String(backendPort) }
delete backendEnvironment.SUBFLOW_SETTINGS_ENCRYPTION_KEY
const backend = start('go', ['run', './cmd/subflow', 'serve', `--env-file=${envFile}`, `--dir=${dataDir}`], {
  cwd: backendDir,
  env: backendEnvironment,
})
backend.stdout.on('data', chunk => { backendOutput = (backendOutput + chunk.toString()).slice(-30000) })
backend.stderr.on('data', chunk => { backendOutput = (backendOutput + chunk.toString()).slice(-30000) })
backend.on('exit', code => { if (!stopping && code !== 0) { console.error(`Isolated backend exited (${code ?? 'signal'}). Raw logs are withheld because they may contain credentials.`); process.exitCode = code || 1 } })

let setupToken
for (let attempt = 0; attempt < 600 && !setupToken && backend.exitCode === null; attempt++) {
    const match = /setup[?]token=([A-Za-z0-9_-]+)/.exec(backendOutput)
  if (match) setupToken = match[1]
  else await new Promise(resolveDelay => setTimeout(resolveDelay, 500))
}
if (!setupToken) {
  const diagnostics = safeBackendDiagnostics()
  stop()
  throw new Error(`Isolated backend did not issue a setup token (exit=${backend.exitCode ?? 'still running'}). Raw logs are withheld because they may contain credentials.${diagnostics ? ` Safe error summary:\n${diagnostics}` : ''}`)
}

const setupStatus = await fetch(`${backendURL}/api/subflow/v1/setup/status?token=${setupToken}`)
if (!(await setupStatus.json()).data?.setupAvailable) { stop(); throw new Error('The isolated backend did not accept its own setup token.') }

const initialized = await fetch(`${backendURL}/api/subflow/v1/setup/initialize`, {
  method: 'POST', headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ token: setupToken, adminName: 'Playwright Owner', email: 'owner@example.test', password: 'test-password-1234', siteName: 'SubFlow E2E', defaultTimezone: 'Asia/Taipei', defaultCurrency: 'TWD', allowRegistration: true }),
})
if (!initialized.ok) { stop(); throw new Error(`Test setup failed (${initialized.status}).`) }

start(process.execPath, [join(frontendDir, 'node_modules', 'vite', 'bin', 'vite.js'), '--host', '127.0.0.1', '--port', '4174', '--strictPort'], {
  cwd: frontendDir,
  env: { ...process.env, SUBFLOW_HTTP_PORT: String(backendPort) },
})

for (let attempt = 0; attempt < 120; attempt++) {
  try {
    const response = await fetch('http://127.0.0.1:4174/api/health')
    if (response.ok) break
  } catch { /* wait for Vite to bind */ }
  await new Promise(resolveDelay => setTimeout(resolveDelay, 500))
  if (attempt === 119) throw new Error('Vite did not become ready on port 4174')
}

console.log('Isolated SubFlow backend and Vite are ready for Playwright.')
await new Promise(resolvePromise => { if (stopping) resolvePromise(); else process.once('beforeExit', resolvePromise) })
