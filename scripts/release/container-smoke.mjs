import assert from 'node:assert/strict'
import { execFileSync } from 'node:child_process'
import { randomUUID } from 'node:crypto'
import { mkdir, writeFile, readFile } from 'node:fs/promises'
import { resolve } from 'node:path'
import { pathToFileURL } from 'node:url'
import { verifySharedAttachment } from './shared-attachment-smoke.mjs'

const root = process.cwd()
const image = process.argv[2] || 'subflow:candidate'
const version = JSON.parse(await readFile(resolve(root, 'frontend/package.json'), 'utf8')).version
const name = `subflow-smoke-${randomUUID()}`
function docker(...args) {
  return execFileSync(process.platform === 'win32' ? 'rtk' : 'docker', process.platform === 'win32' ? ['proxy', 'docker', ...args] : args, { encoding: 'utf8', stdio: ['ignore', 'pipe', 'pipe'] }).trim()
}
await mkdir('.release', { recursive: true })
let started = false
let browser
try {
  docker('run', '--detach', '--rm', '--name', name, '--publish', '127.0.0.1::8080', '--tmpfs', '/app/pb_data', '-e', 'SUBFLOW_ENV=development', '-e', `SUBFLOW_SETTINGS_ENCRYPTION_KEY=${randomUUID()}`, image)
  started = true
  const baseURL = `http://${docker('port', name, '8080/tcp')}`
  let ready = false
  for (let attempt = 0; attempt < 120; attempt++) {
    try { ready = (await fetch(`${baseURL}/api/health`)).ok } catch {}
    if (ready) break
    await new Promise(resolveDelay => setTimeout(resolveDelay, 500))
  }
  assert.ok(ready, 'production container health must become ready')
  const token = /setup[?]token=([A-Za-z0-9_-]+)/.exec(docker('logs', name))?.[1]
  assert.ok(token, 'fresh isolated data must issue a one-time setup token')
  const { chromium, expect } = await import(pathToFileURL(resolve(root, 'frontend/node_modules/@playwright/test/index.mjs')).href)
  browser = await chromium.launch()
  const page = await browser.newPage({ baseURL, viewport: { width: 390, height: 844 }, isMobile: true, hasTouch: true, timezoneId: 'UTC' })
  await page.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))
  const errors = []
  page.on('pageerror', error => errors.push(error.message))
  const owner = 'container-owner@example.test'
  const password = 'isolated-password-1234'
  await page.goto(`/setup?token=${token}`)
  await expect(page.getByRole('heading', { name: 'Create the administrator', exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Complete setup', exact: true }).click()
  assert.equal(await page.locator('form').evaluate(form => form.checkValidity()), false, 'empty setup must fail validation')
  await page.getByLabel('Display name', { exact: true }).fill('Container Owner')
  await page.getByLabel('Email', { exact: true }).fill(owner)
  await page.locator('input[autocomplete="new-password"]').fill(password)
  await page.getByLabel('Allow new user registrations', { exact: true }).check()
  await page.getByRole('button', { name: 'Complete setup', exact: true }).click()
  await expect(page).toHaveURL(/\/home$/, { timeout: 20000 })
  await page.getByRole('button', { name: 'Not now' }).click({ timeout: 1500 }).catch(() => {})
  const auth = async email => {
    const response = await page.request.post('/api/collections/users/auth-with-password', { data: { identity: email, password } })
    assert.equal(response.status(), 200)
    return (await response.json()).token
  }
  const adminToken = await auth(owner)
  const ownerID = await page.evaluate(() => JSON.parse(localStorage.getItem('pocketbase_auth')).record.id)
  await verifySharedAttachment(browser, baseURL, adminToken, ownerID, expect)
  const list = '/api/subflow/v1/system/notification-emails/failed?page=1&perPage=25&sort=-created'
  assert.equal((await page.request.get(list, { headers: { Authorization: adminToken } })).status(), 200)
  await page.goto('/admin/notification-emails')
  await expect(page.getByRole('heading', { name: 'Failed notification emails', exact: true })).toBeVisible()
  await page.reload()
  await expect(page.getByText('No failed notification emails', { exact: true })).toBeVisible()
  await page.screenshot({ path: '.release/admin-mobile.png', fullPage: true })
  await page.goto('/about')
  await expect(page.locator('body')).toContainText(version)
  // Start a second browser context so registration is tested without the owner session.
  const visitor = await browser.newPage({ baseURL, viewport: { width: 1440, height: 900 }, timezoneId: 'UTC' })
  visitor.on('pageerror', error => errors.push(error.message))
  await visitor.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))
  await visitor.goto('/auth')
  await visitor.locator('.segmented').getByRole('button', { name: 'Create account', exact: true }).click()
  await visitor.getByLabel('Display name', { exact: true }).fill('Container Member')
  const member = 'container-member@example.test'
  await visitor.getByLabel('Email', { exact: true }).fill(member)
  await visitor.locator('input[autocomplete="new-password"]').fill(password)
  await visitor.locator('form').getByRole('button', { name: 'Create account', exact: true }).click()
  await expect(visitor.locator('input[autocomplete="current-password"]')).toBeVisible()
  await visitor.locator('input[autocomplete="current-password"]').fill('incorrect-password')
  await visitor.locator('form').getByRole('button', { name: 'Log in', exact: true }).click()
  await expect(visitor.locator('.form-error')).toBeVisible()
  await visitor.locator('input[autocomplete="current-password"]').fill(password)
  await visitor.locator('form').getByRole('button', { name: 'Log in', exact: true }).click()
  await expect(visitor).toHaveURL(/home/)
  const memberToken = await auth(member)
  assert.equal((await page.request.get(list, { headers: { Authorization: memberToken } })).status(), 403)
  await visitor.goto('/admin/notification-emails')
  await expect(visitor.locator('body')).not.toContainText('No failed notification emails')
  await visitor.goto('/personal')
  await visitor.getByRole('button', { name: 'Sign out', exact: true }).click()
  await expect(visitor).toHaveURL(/\/auth/)
  assert.equal((await page.request.post('/api/subflow/v1/auth/logout')).status(), 403)
  const auditResponse = await page.request.get('/api/subflow/v1/system/audit-logs?perPage=50&action=auth.logout', { headers: { Authorization: adminToken } })
  assert.equal(auditResponse.status(), 200)
  const logoutAudits = (await auditResponse.json()).data
  assert.ok(logoutAudits.some(value => value.action === 'auth.logout' && value.outcome === 'success'))
  assert.ok(logoutAudits.some(value => value.action === 'auth.logout' && value.outcome === 'failure'))
  assert.ok(logoutAudits.every(value => !JSON.stringify(value).includes(memberToken)))
  const secondSetup = await page.request.post('/api/subflow/v1/setup/initialize', { data: { token, adminName: 'Second', email: 'second@example.test', password } })
  assert.ok(secondSetup.status() >= 400, 'setup cannot be repeated')
  assert.deepEqual(errors, [], 'no browser runtime errors')
  const result = { version, image, freshSetupUI: true, registrationUI: true, invalidLogin: true, logoutUI: true, logoutBackendAudits: true, sharedAttachmentDialog: true, sharedPdfPreview: true, adminList: 200, regularUserDenied: 403, directReload: true, aboutVersion: true, runtimeErrors: 0, device: 'Chromium touch simulation 390px; desktop 1440px' }
  await writeFile('.release/container-smoke.json', `${JSON.stringify(result, null, 2)}\n`)
  console.log(JSON.stringify(result))
} finally {
  if (browser) await browser.close()
  // Only this invocation's randomly named container owns this tmpfs data.
  if (started) docker('rm', '--force', name)
}
