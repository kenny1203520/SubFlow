import { expect, test } from '@playwright/test'
import { randomUUID } from 'node:crypto'

test('saving an edit remains available during a delayed real SSE refresh', async ({ page }) => {
  const prefix = `save-${randomUUID()}`
  const email = `${prefix}@example.test`
  const registered = await page.request.post('/api/subflow/v1/auth/register', { data: { adminName: 'Ledger save regression', email, password: 'test-password-1234' } })
  expect(registered.status()).toBe(201)
  await page.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))
  await page.goto('/auth')
  await page.getByLabel('Email').fill(email)
  await page.locator('input[autocomplete="current-password"]').fill('test-password-1234')
  await page.locator('form').getByRole('button', { name: 'Log in' }).click()
  await expect(page).toHaveURL(/home/)
  await page.getByRole('button', { name: 'Not now' }).click({ timeout: 1500 }).catch(() => {})
  await page.goto('/personal')
  const token = await page.evaluate(() => JSON.parse(localStorage.getItem('pocketbase_auth') || '{}').token as string)
  const headers = { Authorization: `Bearer ${token}` }
  await page.getByRole('button', { name: 'Add record', exact: true }).click()
  let drawer = page.getByRole('dialog', { name: 'Add record' })
  await drawer.getByLabel('Item').fill(`${prefix} original`)
  await drawer.getByLabel('Amount').fill('123')
  const created = page.waitForResponse(response => new URL(response.url()).pathname === '/api/subflow/v1/expenses' && response.request().method() === 'POST')
  await drawer.getByRole('button', { name: 'Save record' }).click()
  const record = (await (await created).json()).data
  await expect(drawer).toBeHidden()
  const row = page.locator('.ledger-item').filter({ hasText: `${prefix} original` })
  await expect(row).toHaveCount(1)
  await row.getByRole('button', { name: 'Edit' }).click()
  drawer = page.getByRole('dialog', { name: 'Edit expense' })
  await expect(drawer.getByLabel('Amount')).toHaveValue('123.00')
  await drawer.getByLabel('Item').fill(`${prefix} updated`)

  let held = false, released = false
  let resume: () => void = () => {}
  const gate = new Promise<void>(resolve => { resume = resolve })
  const pattern = '**/api/subflow/v1/expenses?**'
  await page.route(pattern, async route => {
    if (!held && !released) { held = true; await gate }
    await route.continue()
  })
  try {
    // A real mutation emits the SSE event. Only the following read is delayed.
    const date = await page.getByLabel('Choose date').inputValue()
    const background = await page.request.post('/api/subflow/v1/expenses', { headers, data: { title: `${prefix} background`, amountMinor: 100, currency: 'TWD', incurredOn: `${date}T00:00:00Z`, paidBy: record.paidBy } })
    expect(background.status()).toBe(201)
    await expect.poll(() => held).toBe(true)
    const save = drawer.getByRole('button', { name: 'Save record' })
    await expect(save).toBeEnabled()
    const patched = page.waitForResponse(response => new URL(response.url()).pathname === `/api/subflow/v1/expenses/${record.id}` && response.request().method() === 'PATCH', { timeout: 10_000 })
    await save.click()
    expect((await patched).status()).toBe(200)
    released = true; resume()
    await expect(drawer).toBeHidden()
    await expect(page.locator('.ledger-item').filter({ hasText: `${prefix} updated` })).toHaveCount(1)
    const records = await page.request.get('/api/subflow/v1/expenses?scope=all&perPage=50', { headers })
    expect((await records.json()).data.filter((value: { id: string }) => value.id === record.id)).toEqual([expect.objectContaining({ id: record.id, title: `${prefix} updated` })])
  } finally { released = true; resume(); await page.unrouteAll({ behavior: 'wait' }) }
})
