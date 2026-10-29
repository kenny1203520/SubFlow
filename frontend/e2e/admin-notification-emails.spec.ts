import { expect, test, type Page } from '@playwright/test'
import { randomUUID } from 'node:crypto'

async function signIn(page: Page, email = 'owner@example.test') {
  await page.goto('/auth')
  await page.evaluate(() => localStorage.setItem('subflow-locale', 'en'))
  await page.reload()
  await page.getByLabel('Email').fill(email)
  await page.locator('input[autocomplete="current-password"]').fill('test-password-1234')
  await page.locator('form').getByRole('button', { name: 'Log in' }).click()
  await expect(page).toHaveURL(/home/)
  await page.getByRole('button', { name: 'Not now' }).click({ timeout: 1500 }).catch(() => {})
}

test('failed notification emails use the real API and survive a direct reload', async ({ page }) => {
  await signIn(page)
  await page.goto('/admin')
  const responsePromise = page.waitForResponse(response => new URL(response.url()).pathname === '/api/subflow/v1/system/notification-emails/failed')
  await page.getByRole('link', { name: 'Failed notification emails' }).click()
  const response = await responsePromise
  expect(response.status(), await response.text()).toBe(200)
  await expect(page.getByRole('heading', { name: 'Failed notification emails', exact: true })).toBeVisible()
  await page.reload()
  await expect(page).toHaveURL(/\/admin\/notification-emails$/)
  await expect(page.getByRole('heading', { name: 'Failed notification emails', exact: true })).toBeVisible()
})

test('a simulated unavailable endpoint shows an error and retries the real backend', async ({ page }) => {
  await signIn(page)
  let unavailable = true
  await page.route('**/system/notification-emails/failed?**', async route => {
    if (unavailable) await route.fulfill({ status: 404, json: { error: { code: 'not_found', message: 'Not found' } } })
    else await route.continue()
  })
  const requests: string[] = []
  page.on('request', request => { const path = new URL(request.url()).pathname; if (path.includes('/system/')) requests.push(path) })
  await page.goto('/admin/notification-emails')
  const alert = page.getByRole('alert')
  await expect(alert).toContainText('Notification email management is unavailable')
  await expect(page.getByText('No failed notification emails', { exact: true })).toBeHidden()
  expect(requests.filter(path => !path.endsWith('/access') && !path.endsWith('/notification-emails/failed'))).toEqual([])
  unavailable = false
  const responsePromise = page.waitForResponse(response => new URL(response.url()).pathname.endsWith('/notification-emails/failed') && response.status() === 200)
  await alert.getByRole('button', { name: 'Retry', exact: true }).click()
  await responsePromise
  await expect(alert).toBeHidden()
  await expect(page.getByText('No failed notification emails', { exact: true })).toBeVisible()
})

test('simulated long email jobs wrap, paginate, and retain their row after a real rejected retry', async ({ page }, testInfo) => {
  await signIn(page)
  const longCode = 'mail_delivery_failed_很長的錯誤代碼_'.repeat(18)
  await page.route('**/system/notification-emails/failed?**', async route => {
    const query = new URL(route.request().url()).searchParams
    const pageNumber = Number(query.get('page') || 1), perPage = Number(query.get('perPage') || 25)
    await route.fulfill({ json: { data: [{ id: 'missing-e2e-job', notificationId: 'simulated', status: 'failed', attempts: 5, lastError: longCode, createdAt: '2026-10-22T00:00:00Z' }], meta: { page: pageNumber, perPage, totalItems: 26, totalPages: Math.ceil(26 / perPage) } } })
  })
  await page.goto('/admin/notification-emails')
  const row = page.locator('.failed-email-row')
  await expect(row).toContainText(longCode)
  for (const width of [320, 375, 390, 768, 1024, 1440]) {
    await page.setViewportSize({ width, height: 900 })
    const sizes = await page.evaluate(() => ({ viewport: document.documentElement.clientWidth, content: document.documentElement.scrollWidth }))
    expect(sizes.content).toBeLessThanOrEqual(sizes.viewport + 1)
    await expect.poll(() => row.locator('small').evaluate(element => getComputedStyle(element).whiteSpace)).toBe('normal')
    await expect.poll(() => page.locator('.admin-tabs').evaluate(nav => {
      const active = nav.querySelector('[aria-current="page"]')?.getBoundingClientRect(), bounds = nav.getBoundingClientRect()
      return !!active && active.left >= bounds.left - 1 && active.right <= bounds.right + 1
    })).toBe(true)
    if (width === 320 || width === 1440) await page.screenshot({ path: testInfo.outputPath(`failed-emails-${width}.png`), fullPage: true, animations: 'disabled' })
  }
  await page.getByRole('button', { name: 'Next', exact: true }).click()
  await expect(page).toHaveURL(/page=2/)
  await page.getByLabel('Per page').selectOption('10')
  await expect(page).toHaveURL(/perPage=10/)
  await expect(page).not.toHaveURL(/page=2/)
  const retryResponse = page.waitForResponse(response => response.url().includes('/notification-emails/missing-e2e-job/retry'))
  await row.getByRole('button', { name: 'Queue retry', exact: true }).click()
  const rejected = await retryResponse
  expect(rejected.status()).toBe(404)
  await expect(page.getByRole('alert')).toBeVisible()
  await expect(row).toBeVisible()
  await expect(page.getByText('The notification email has been queued again.')).toBeHidden()
  const headers = await rejected.request().allHeaders()
  const audit = await page.request.get('/api/subflow/v1/system/audit-logs?action=notification.email.retried&outcome=failure&perPage=100', { headers: { Authorization: headers.authorization || '' } })
  expect(audit.status()).toBe(200)
  expect((await audit.json()).data).toEqual(expect.arrayContaining([expect.objectContaining({ action: 'notification.email.retried', resourceId: 'missing-e2e-job', outcome: 'failure' })]))
})

test('a regular account cannot access notification email management or its API', async ({ page }) => {
  const email = `email-permission-${randomUUID()}@example.test`
  const registration = await page.request.post('/api/subflow/v1/auth/register', { data: { adminName: 'Regular account', email, password: 'test-password-1234' } })
  expect(registration.status()).toBe(201)
  await signIn(page, email)
  const token = await page.evaluate(() => JSON.parse(localStorage.getItem('pocketbase_auth') || '{}').token as string)
  const denied = await page.request.get('/api/subflow/v1/system/notification-emails/failed?page=1&perPage=25&sort=-created', { headers: { Authorization: `Bearer ${token}` } })
  expect(denied.status()).toBe(403)
  await page.goto('/admin/notification-emails')
  await expect(page).toHaveURL(/\/home\?denied=admin$/)
  await expect(page.getByRole('heading', { name: 'Failed notification emails' })).toBeHidden()
})
