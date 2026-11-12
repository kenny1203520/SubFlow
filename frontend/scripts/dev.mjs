import { spawn } from 'node:child_process'
import { existsSync, readFileSync } from 'node:fs'
import { resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { selectViteEnvironment } from './dev-environment.mjs'

const frontendDir = resolve(fileURLToPath(new URL('..', import.meta.url)))
const repositoryRoot = resolve(frontendDir, '..')
const developmentEnvFile = resolve(repositoryRoot, '.env.development.local')
const developmentEnvContents = existsSync(developmentEnvFile) ? readFileSync(developmentEnvFile, 'utf8') : undefined
const viteEnvironment = selectViteEnvironment(process.env, developmentEnvContents)

const viteCLI = resolve(frontendDir, 'node_modules', 'vite', 'bin', 'vite.js')
const vite = spawn(process.execPath, [viteCLI, ...process.argv.slice(2)], {
  cwd: frontendDir,
  env: viteEnvironment,
  stdio: 'inherit',
})

for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, () => vite.kill(signal))
}

vite.on('error', error => {
  console.error(`Could not start Vite: ${error.message}`)
  process.exitCode = 1
})

vite.on('close', (code, signal) => {
  process.exitCode = code ?? (signal === 'SIGINT' ? 130 : signal ? 1 : 0)
})
