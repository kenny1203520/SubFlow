import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { randomUUID } from 'node:crypto'
import { mkdir, writeFile } from 'node:fs/promises'

const oldImage = process.argv[2] || 'ghcr.io/kenny1203520/subflow@sha256:99bcb63a7322dc66feb3fc36bb6840cf727b1cc1e283e66e4f83d284a9bebc03'
const candidate = process.argv[3] || 'subflow:candidate'
const sourceVersion = process.argv[4] || 'alpha1'
assert.match(sourceVersion, /^alpha[1-5]$/)
const prefix = `subflow-upgrade-${randomUUID()}`
const volumes = new Set()
const containers = new Set()
const key = randomUUID()
const password = 'isolated-upgrade-password-1234'
const owner = 'upgrade-owner@example.test'
const member = 'upgrade-member@example.test'
function docker(...args) {
  return execFileSync(process.platform === 'win32' ? 'rtk' : 'docker', process.platform === 'win32' ? ['proxy', 'docker', ...args] : args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim()
}
function safeUpgradeDetail(name) {
  try {
    const line = docker('logs', name).split(/\r?\n/).reverse().find(value => /SubFlow upgrade stopped:|cannot prepare upgrade status:/i.test(value))
    if (!line) return 'detail-unavailable'
    return line
      .replace(/^.*SubFlow upgrade stopped:\s*/i, '')
      .replace(/((?:token|secret|key|password)\s*[:=]\s*)\S+/gi, '$1[redacted]')
      .replace(/Bearer\s+\S+/gi, 'Bearer [redacted]')
      .replace(/\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b/gi, '[email]')
      .replace(/[A-Za-z]:\\[^\s:]+/g, '[path]')
      .replace(/\/(?:[^\s/:]+\/)*[^\s/:]+/g, '[path]')
      .slice(0, 180)
  } catch { return 'detail-unavailable' }
}
function volume(suffix) {
  const name = `${prefix}-${suffix}`
  docker('volume', 'create', name)
  volumes.add(name)
  return name
}
async function start(image, data, suffix) {
  const name = `${prefix}-${suffix}`
  docker('run', '--detach', '--rm', '--name', name, '--publish', '127.0.0.1::8080', '--volume', `${data}:/app/pb_data`, '-e', 'SUBFLOW_ENV=development', '-e', `SUBFLOW_SETTINGS_ENCRYPTION_KEY=${key}`, image)
  containers.add(name)
  const base = `http://${docker('port', name, '8080/tcp')}`
  let healthStatus = 'unreachable'
  let runtimeStatus = 'unavailable'
  for (let attempt = 0; attempt < 600; attempt++) {
    try {
      const response = await fetch(`${base}/api/health`)
      healthStatus = `http-${response.status}`
      if (response.ok) return { name, base }
    } catch { healthStatus = 'unreachable' }
    if (attempt % 5 === 0) {
      try {
        const response = await fetch(`${base}/api/subflow/v1/runtime/status`)
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
  const response = await fetch(`${server.base}${path}`, { method, headers: { ...(token ? { Authorization: `Bearer ${token}` } : {}), ...(body && !form ? { 'Content-Type': 'application/json' } : {}) }, body: body ? form ? body : JSON.stringify(body) : undefined })
  assert.ok(response.ok, `${method} ${path.split('?')[0]} returned ${response.status}`)
  return response
}
async function api(server, path, token, method, body) { return (await (await request(server, `/api/subflow/v1${path}`, token, method, body)).json()).data }
async function login(server, email) { return (await (await request(server, '/api/collections/users/auth-with-password', '', 'POST', { identity: email, password })).json()).token }
function copyVolume(from, to) {
  // Both volumes were created by this invocation. Copy only while the app is
  // stopped, and keep the backup read-only during restore.
  assert.ok(volumes.has(from) && volumes.has(to))
  docker('run', '--rm', '--entrypoint', '/bin/sh', '--volume', `${from}:/from:ro`, '--volume', `${to}:/to`, oldImage, '-c', 'cp -a /from/. /to/')
}

try {
  const data = volume('data'), backup = volume('backup'), restored = volume('restored')
  let server = await start(oldImage, data, 'old')
  const setupToken = /setup[?]token=([A-Za-z0-9_-]+)/.exec(docker('logs', server.name))?.[1]
  assert.ok(setupToken)
  await api(server, '/setup/initialize', '', 'POST', { token: setupToken, adminName: 'Upgrade Owner', email: owner, password, siteName: 'Upgrade fixture', defaultTimezone: 'UTC', defaultCurrency: 'TWD', allowRegistration: true })
  let ownerToken = await login(server, owner)
  await api(server, '/auth/register', '', 'POST', { adminName: 'Upgrade Member', email: member, password })
  let memberToken = await login(server, member)
  const group = await api(server, '/groups', ownerToken, 'POST', { name: 'Preserved group', currency: 'TWD', timezone: 'UTC' })
  const income = await api(server, '/incomes', ownerToken, 'POST', { title: 'Preserved income', amountMinor: 12345, currency: 'TWD', receivedOn: '2026-10-21T00:00:00Z' })
  await api(server, `/groups/${group.id}/invitations`, ownerToken, 'POST', { email: member })
  const notes = await api(server, '/notifications?perPage=50', memberToken)
  assert.ok(notes.length > 0, 'old version must create a legacy invitation notification')
  const form = new FormData()
  form.set('targetType', 'income'); form.set('targetId', income.id); form.set('requestId', randomUUID())
  const file = '%PDF-1.7\npreserved attachment\n%%EOF'
  form.set('file', new Blob([file], { type: 'application/pdf' }), 'preserved.pdf')
  const attachment = await api(server, '/attachments', ownerToken, 'POST', form)
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
  for (const note of upgraded) {
    assert.equal(note.category, 'task')
    assert.equal(note.status, 'pending')
    assert.ok(note.viewedAt, 'legacy notices must not appear as newly unseen')
  }
  const first = upgraded.map(({ id, category, status, viewedAt }) => ({ id, category, status, viewedAt }))
  stop(server)
  server = await start(candidate, data, 'restart')
  upgraded = await verifyData(server)
  assert.deepEqual(upgraded.map(({ id, category, status, viewedAt }) => ({ id, category, status, viewedAt })), first, 'schema backfill must be idempotent on restart')
  stop(server)

  copyVolume(backup, restored)
  server = await start(oldImage, restored, 'restore')
  await verifyData(server)
  stop(server)
  const result = { sourceVersion, oldImage, candidate, isolatedData: true, offlineBackup: true, financialAmountAndIds: true, groupPreserved: true, attachmentBytes: true, notificationBackfill: true, restartIdempotent: true, oldVersionRestore: true }
  await mkdir('.release', { recursive: true })
  await writeFile(`.release/upgrade-smoke-${sourceVersion}.json`, `${JSON.stringify(result, null, 2)}\n`)
  console.log(JSON.stringify(result))
} finally {
  for (const name of containers) { assert.ok(name.startsWith(`${prefix}-`)); docker('rm', '--force', name) }
  for (const name of volumes) { assert.ok(name.startsWith(`${prefix}-`)); docker('volume', 'rm', name) }
}
