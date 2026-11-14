import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { randomUUID } from 'node:crypto'
import { mkdir, mkdtemp, rm, writeFile } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import https from 'node:https'

const oldImage = process.argv[2] || 'ghcr.io/kenny1203520/subflow@sha256:99bcb63a7322dc66feb3fc36bb6840cf727b1cc1e283e66e4f83d284a9bebc03'
const candidate = process.argv[3] || 'subflow:candidate'
const sourceVersion = process.argv[4] || 'alpha1'
assert.match(sourceVersion, /^(alpha[1-5]|beta1?)$/)
const prefix = `subflow-upgrade-${randomUUID()}`
const volumes = new Set()
const mounts = new Set()
const bindRoot = process.env.SUBFLOW_REHEARSAL_BIND === "1" ? await mkdtemp(join(tmpdir(), prefix+"-")) : null
const containers = new Set()
const key = randomUUID()
const password = 'isolated-upgrade-password-1234'
const owner = 'upgrade-owner@example.test'
const member = 'upgrade-member@example.test'
function docker(...args) {
  return execFileSync(process.platform === 'win32' ? 'rtk' : 'docker', process.platform === 'win32' ? ['proxy', 'docker', ...args] : args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim()
}
function requestHTTPS(server, url, options = {}) {
  return new Promise((resolve, reject) => {
    const body = options.body === undefined ? undefined : Buffer.from(options.body)
    const headers = { ...options.headers }
    const requestPath = new URL(url).pathname
    if (body) headers['Content-Length'] = String(body.length)
    const request = https.request(url, { method: options.method || 'GET', headers, ca: server.certificate }, response => {
      const chunks = []
      response.on('data', chunk => chunks.push(chunk))
      response.once('aborted', () => reject(new Error(`HTTPS upgrade response was interrupted: ${requestPath}`)))
      response.once('error', reject)
      response.on('end', () => {
        const text = Buffer.concat(chunks).toString('utf8')
        const status = response.statusCode || 0
        resolve({
          status,
          ok: status >= 200 && status < 300,
          async json() { return JSON.parse(text) },
          async text() { return text },
        })
      })
    })
    request.once('error', reject)
    request.setTimeout(options.timeout || 15_000, () => request.destroy(new Error(`HTTPS upgrade request timed out: ${requestPath}`)))
    if (body) request.write(body)
    request.end()
  })
}
function safeUpgradeDetail(name) {
  const details = []
  try {
    const state = JSON.parse(docker('exec', name, '/bin/sh', '-c', 'cat /app/pb_data/.subflow/state.json'))
    const phase = state.failurePhase || state.phase
    if (typeof phase === 'string' && /^[a-z_]{1,32}$/.test(phase)) details.push(`phase=${phase}`)
    const history = Array.isArray(state.history)
      ? state.history
          .filter(item => typeof item?.id === 'string' && /^[a-z0-9_]{1,64}$/.test(item.id) && ['running', 'failed', 'complete'].includes(item.status))
          .map(item => `${item.id}:${item.status}`)
      : []
    if (history.length) details.push(`migrations=${history.join(',')}`)
  } catch { /* Status may be unavailable if the process stopped before writing it. */ }
  try {
    const lines = docker('logs', name).split(/\r?\n/).filter(value => /error|failed|stopped|denied|no such|migration|backup|unknown schema|audit/i.test(value))
    if (lines.length > 0) details.push(lines.slice(-3).map(line => line
      .replace(/([?&](?:token|key|password|secret)=)[^&\s]+/gi, '$1[redacted]')
      .replace(/((?:token|secret|key|password)\s*[:=]\s*)\S+/gi, '$1[redacted]')
      .replace(/Bearer\s+\S+/gi, 'Bearer [redacted]')
      .replace(/\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b/gi, '[email]')
      .replace(/[A-Za-z]:\\[^\s:]+/g, '[path]')
      .replace(/\/(?:[^\s/:]+\/)*[^\s/:]+/g, '[path]')
      .slice(0, 180)).join(' | '))
  } catch { details.push('docker-logs-unavailable') }
  return details.join('; ') || 'startup-details-unavailable'
}
async function volume(suffix) {
  if (bindRoot) {
    const path = join(bindRoot,suffix)
    await mkdir(path)
    mounts.add(path)
    return path
  }
  const name = `${prefix}-${suffix}`
  docker('volume', 'create', name)
  volumes.add(name)
  return name
}
function launch(image, data, suffix, publishedPort = '127.0.0.1::8080') {
  const name = `${prefix}-${suffix}`
  if (image === candidate) {
    // Historical volumes may be root-owned after older releases. Fix only
    // this isolated rehearsal volume before the nonroot candidate starts.
    docker('run', '--rm', '--user', '0:0', '--volume', `${data}:/app/pb_data`, image, 'init-permissions')
  }
  docker('run', '--detach', '--rm', '--name', name, '--publish', publishedPort, '--volume', `${data}:/app/pb_data`, '-e', 'SUBFLOW_ENV=development', '-e', `SUBFLOW_SETTINGS_ENCRYPTION_KEY=${key}`, image)
  containers.add(name)
  return name
}
async function start(image, data, suffix, publishedPort) {
  const name = launch(image, data, suffix, publishedPort)
  const secure = image === candidate || sourceVersion.startsWith('beta')
  const base = `${secure ? 'https' : 'http'}://${docker('port', name, '8080/tcp')}`
  let certificate
  let healthStatus = 'unreachable'
  let runtimeStatus = 'unavailable'
  for (let attempt = 0; attempt < 600; attempt++) {
    try {
      if (secure && !certificate) {
        const tlsDir = '/app/pb_data/.subflow/tls'
        const active = docker('exec', name, 'cat', `${tlsDir}/active`)
        if (!/^[A-Za-z0-9_-]{1,64}$/.test(active)) throw new Error('invalid active TLS certificate selection')
        const versionDir = active === 'self-signed' ? `${tlsDir}/${active}` : `${tlsDir}/versions/${active}`
        certificate = Buffer.from(docker('exec', name, 'cat', `${versionDir}/cert.pem`))
      }
      const response = secure
        ? await requestHTTPS({ certificate }, `${base}/api/health`, { timeout: 3_000 })
        : await fetch(`${base}/api/health`)
      healthStatus = `${secure ? 'https' : 'http'}-${response.status}`
      if (response.ok) return { name, base, certificate }
    } catch { healthStatus = 'unreachable' }
    if (attempt % 5 === 0) {
      try {
        const response = secure
          ? await requestHTTPS({ certificate }, `${base}/api/subflow/v1/runtime/status`, { timeout: 3_000 })
          : await fetch(`${base}/api/subflow/v1/runtime/status`)
        if (response.ok) {
          const payload = await response.json()
          const state = payload?.data?.state
          const problemCode = payload?.data?.problemCode
          if (['checking', 'backing_up', 'migrating', 'verifying', 'ready', 'failed'].includes(state)) {
            runtimeStatus = state
            if (state === 'failed' && typeof problemCode === 'string' && /^[a-z0-9_]{1,64}$/.test(problemCode)) runtimeStatus += `:${problemCode}`
          }
        }
      } catch { /* Keep the diagnostic limited to public, safe runtime state. */ }
      if (runtimeStatus.startsWith('failed')) {
        throw new Error(`Upgrade test container entered maintenance failure (${healthStatus}, runtime=${runtimeStatus}, cause=${safeUpgradeDetail(name)}).`)
      }
    }
    await new Promise(resolve => setTimeout(resolve, 1000))
  }
  throw new Error(`Upgrade test container did not become healthy (${healthStatus}, runtime=${runtimeStatus}).`)
}
function stop(server) { docker('stop', server.name); containers.delete(server.name) }
async function request(server, path, token, method = 'GET', body) {
  const form = body instanceof FormData
  console.log(`Checking ${method} ${path.split('?')[0]}`)
  const options = { method, headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}), ...(body && !form ? { 'Content-Type': 'application/json' } : {}) }, body: body ? form ? body : JSON.stringify(body) : undefined }
  if (server.certificate && form) {
    const encoded = new Response(body)
    options.headers['Content-Type'] = encoded.headers.get('content-type')
    options.body = await encoded.arrayBuffer()
  }
  const response = server.certificate
    ? await requestHTTPS(server, `${server.base}${path}`, options)
    : await fetch(`${server.base}${path}`, options)
  assert.ok(response.ok, `${method} ${path.split('?')[0]} returned ${response.status}`)
  return response
}
async function api(server, path, token, method, body) { return (await (await request(server, `/api/subflow/v1${path}`, token, method, body)).json()).data }
async function login(server, email) { return (await (await request(server, '/api/collections/users/auth-with-password', '', 'POST', { identity: email, password })).json()).token }
function copyVolume(from, to) {
  // Both volumes were created by this invocation. Copy only while the app is
  // stopped, and keep the backup read-only during restore.
  assert.ok((volumes.has(from) || mounts.has(from)) && (volumes.has(to) || mounts.has(to)))
  docker('run', '--rm', '--user', '0:0', '--entrypoint', '/bin/sh',
    '--volume', `${from}:/from:ro`, '--volume', `${to}:/to`, oldImage, '-c', 'cp -a /from/. /to/')
}

try {
  const data = await volume('data'), backup = await volume('backup'), restored = await volume('restored')
  let server = await start(oldImage, data, 'old')
  const setupToken = /setup[?]token=([A-Za-z0-9_-]+)/.exec(docker('logs', server.name))?.[1]
  assert.ok(setupToken)
  await api(server, '/setup/initialize', '', 'POST', { token: setupToken, adminName: 'Upgrade Owner', email: owner, password, siteName: 'Upgrade fixture', defaultTimezone: 'UTC', defaultCurrency: 'TWD', allowRegistration: true })
  let ownerToken = await login(server, owner)
  await api(server, '/auth/register', '', 'POST', { adminName: 'Upgrade Member', email: member, password })
  let memberToken = await login(server, member)
  const group = await api(server, '/groups', ownerToken, 'POST', { name: 'Preserved group', currency: 'TWD', timezone: 'UTC' })
  const income = await api(server, '/incomes', ownerToken, 'POST', { title: 'Preserved income', amountMinor: 12345, currency: 'TWD', receivedOn: '2026-10-21T00:00:00Z' })
  // Alpha fixtures exercise the legacy invitation backfill. Beta invitations
  // additionally require a configured public URL, unrelated to this rehearsal.
  if (sourceVersion.startsWith('alpha')) {
    await api(server, `/groups/${group.id}/invitations`, ownerToken, 'POST', { email: member })
  }
  const notes = await api(server, '/notifications?perPage=50', memberToken)
  if (sourceVersion.startsWith('alpha')) {
    assert.ok(notes.length > 0, 'old version must create a legacy invitation notification')
  }
  const form = new FormData()
  form.set('targetType', 'income'); form.set('targetId', income.id); form.set('requestId', randomUUID())
  const file = '%PDF-1.7\npreserved attachment\n%%EOF'
  form.set('file', new Blob([file], { type: 'application/pdf' }), 'preserved.pdf')
  const attachment = await api(server, '/attachments', ownerToken, 'POST', form)
  if (sourceVersion === 'beta1') {
    docker('exec', server.name, '/bin/sh', '-c',
      'mkdir -p /app/pb_data/backups; printf old-daily > /app/pb_data/backups/daily-fixture-1.zip; printf old-daily > /app/pb_data/backups/daily-fixture-2.zip')
  }
  stop(server)
  copyVolume(data, backup)

  const verifyData = async current => {
    ownerToken = await login(current, owner)
    memberToken = await login(current, member)
    const incomes = await api(current, '/incomes?perPage=50', ownerToken)
    const record = incomes.find(value => value.id === income.id)
    assert.equal(record?.amountMinor, 12345, 'financial amount and ID must survive')
    const groups = await api(current, '/groups', ownerToken)
    assert.ok(groups.some(value => value.id === group.id && value.name === 'Preserved group'))
    const content = await request(current, `/api/subflow/v1/attachments/${attachment.id}/content`, ownerToken)
    assert.equal(await content.text(), file, 'attachment bytes must survive upgrade and restore')
    const notifications = await api(current, '/notifications?perPage=50', memberToken)
    assert.deepEqual(notifications.map(value => value.id).sort(), notes.map(value => value.id).sort(), 'legacy notification IDs must survive without duplicates')
    return notifications
  }
  server = await start(candidate, data, 'new')
  let upgraded = await verifyData(server)
  const status = JSON.parse(docker('exec', server.name, '/app/subflow', 'upgrade', 'status', '--dir=/app/pb_data'))
  assert.equal(status.state.ready, true)
  assert.equal(status.state.backupId, undefined)
  assert.equal(status.backups.filter(item => item.status === 'successful').length, 0)
  if (sourceVersion === 'beta1') {
    for (const archive of ['daily-fixture-1.zip', 'daily-fixture-2.zip']) {
      assert.equal(docker('exec', server.name, 'cat', '/app/pb_data/backups/' + archive), 'old-daily')
    }
  }
  if (sourceVersion === 'alpha1') {
    for (const note of upgraded) {
      assert.equal(note.category, 'task')
      assert.equal(note.status, 'pending')
      assert.ok(note.viewedAt, 'alpha.1 notices predating the unseen marker must not appear as newly unseen')
    }
  } else {
    const originalState = notes.map(({ id, category, status, viewedAt }) => ({ id, category, status, viewedAt }))
    const migratedState = upgraded.map(({ id, category, status, viewedAt }) => ({ id, category, status, viewedAt }))
    assert.deepEqual(migratedState, originalState, 'alpha.2+ notification lifecycle state must be preserved')
  }
  const first = upgraded.map(({ id, category, status, viewedAt }) => ({ id, category, status, viewedAt }))
  stop(server)
  server = await start(candidate, data, 'restart')
  upgraded = await verifyData(server)
  assert.deepEqual(upgraded.map(({ id, category, status, viewedAt }) => ({ id, category, status, viewedAt })), first, 'schema backfill must be idempotent on restart')
  stop(server)

  if (sourceVersion === 'beta1') {
    const maintenance = (image, ...args) => docker('run', '--rm', '--volume', `${data}:/app/pb_data`,
      '-e', 'SUBFLOW_ENV=development', '-e', `SUBFLOW_SETTINGS_ENCRYPTION_KEY=${key}`, image, 'upgrade', ...args, '--dir=/app/pb_data')
    assert.throws(() => maintenance(oldImage, 'retry'), error => {
      assert.match(String(error.stderr), /downgrades are not supported/)
      return error.status !== 0
    })
    const failed = JSON.parse(maintenance(candidate, 'status', '--state-only'))
    assert.equal(failed.problemCode, 'unsupported_downgrade')
    assert.throws(() => docker('run', '--rm', '--entrypoint', '/app/subflow',
      '--volume', `${data}:/app/pb_data:ro`, candidate, 'upgrade', 'preflight', '--dir=/app/pb_data'), error => {
      const report = JSON.parse(String(error.stdout))
      assert.deepEqual(report.blockers, ['manual_retry_required'])
      return error.status !== 0
    })
    const maintenanceName = launch(candidate, data, 'maintenance')
    const address = docker('port', maintenanceName, '8080/tcp')
    const { chromium, expect } = await import(new URL('../../frontend/node_modules/@playwright/test/index.mjs', import.meta.url).href)
    const browser = await chromium.launch()
    try {
      const page = await browser.newPage({ ignoreHTTPSErrors: true })
      await expect.poll(async () => {
        try {
          const response = await page.request.get(`https://${address}/api/subflow/v1/runtime/status`, { timeout: 2000 })
          return (await response.json()).data?.state
        } catch { return undefined }
      }, { timeout: 30000 }).toBe('failed')
      await page.goto(`https://${address}/auth`)
      const dialog = page.getByRole('dialog')
      await expect(dialog).toContainText('unsupported_downgrade')
      stop({ name: maintenanceName })
      assert.match(maintenance(candidate, 'retry'), /upgrade retry completed/)
      const retried = JSON.parse(maintenance(candidate, 'status', '--state-only'))
      assert.equal(retried.ready, true)
      assert.equal(retried.firstFailure.code, 'unsupported_downgrade')
      server = await start(candidate, data, 'retried', `${address}:8080`)
      await expect(dialog).toBeHidden({ timeout: 40000 })
      await expect(page.locator('input[autocomplete="current-password"]')).toBeVisible()
    } finally {
      await browser.close()
    }
    for (let attempt = 0; attempt < 40; attempt++) {
      if (docker('inspect', '--format', '{{.State.Health.Status}}', server.name) === 'healthy') break
      await new Promise(resolve => setTimeout(resolve, 1000))
    }
    assert.equal(docker('inspect', '--format', '{{.State.Health.Status}}', server.name), 'healthy')
    await verifyData(server)
    stop(server)
  }

  copyVolume(backup, restored)
  server = await start(oldImage, restored, 'restore')
  await verifyData(server)
  stop(server)
  const result = { sourceVersion, oldImage, candidate, isolatedData: true, offlineBackup: true, financialAmountAndIds: true, groupPreserved: true, attachmentBytes: true, notificationStatePreserved: true, restartIdempotent: true, oldVersionRestore: true }
  await mkdir('.release', { recursive: true })
  await writeFile(`.release/upgrade-smoke-${sourceVersion}.json`, `${JSON.stringify(result, null, 2)}\n`)
  console.log(JSON.stringify(result))
} finally {
  for (const name of containers) { assert.ok(name.startsWith(`${prefix}-`)); docker('rm', '--force', name) }
  if (bindRoot) await rm(bindRoot, {recursive:true,force:true})
  for (const name of volumes) { assert.ok(name.startsWith(`${prefix}-`)); docker('volume', 'rm', name) }
}
