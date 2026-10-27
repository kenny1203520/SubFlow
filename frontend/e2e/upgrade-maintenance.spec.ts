import { expect, test } from '@playwright/test'

test('announces maintenance accessibly and resumes when the backend is ready', async ({ page }) => {
  let phase = 'migrating'
  let ready = false
  await page.route('**/api/subflow/v1/runtime/status', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ data: { state: phase, phase, ready } }),
  }))
  await page.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))

  await page.goto('/')
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await expect(dialog).toHaveAttribute('open', '')
  await expect(dialog.getByRole('status')).toContainText('Updating the data schema', { timeout: 10000 })

  phase = 'ready'
  ready = true
  await expect(dialog).toBeHidden({ timeout: 10000 })
  await expect(page.locator('.app-content')).not.toHaveAttribute('inert', '')
})

test('shows the safe maintenance problem code without exposing server details', async ({ page }) => {
  await page.route('**/api/subflow/v1/runtime/status', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ data: { state: 'failed', phase: 'failed', ready: false, problemCode: 'backup_failed', detail: 'C:\\private\\data.db' } }),
  }))
  await page.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))

  await page.goto('/')
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await expect(dialog).toHaveAttribute('open', '')
  await expect(dialog).toContainText('backup_failed')
  await expect(dialog).not.toContainText('private')
})

test('asks an incompatible frontend to reload only after the user handles unsaved content', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))
  await page.route('**/api/subflow/v1/runtime/status', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ data: { state: 'ready', phase: 'ready', ready: true } }),
  }))
  await page.route('**/api/subflow/v1/setup/status*', route => route.fulfill({
    status: 426,
    contentType: 'application/json',
    body: JSON.stringify({ error: { code: 'client_update_required', message: 'Update required' } }),
  }))

  await page.goto('/')
  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await expect(dialog.getByRole('heading', { name: 'Update the SubFlow frontend' })).toBeVisible()
  await expect(dialog.getByRole('button', { name: 'I handled unsaved content; reload' })).toBeVisible()
})

test('keeps an open login form mounted while maintenance pauses the application', async ({ page }) => {
  let phase = 'ready'
  let ready = true
  await page.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))
  await page.route('**/api/subflow/v1/runtime/status', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify({ data: { state: phase, phase, ready } }),
  }))

  await page.goto('/auth')
  const email = page.getByLabel('Email', { exact: true })
  const password = page.locator('input[autocomplete="current-password"]')
  await email.fill('not-yet-saved@example.test')
  await password.fill('unsaved-password')
  phase = 'migrating'
  ready = false
  await page.evaluate(() => window.dispatchEvent(new CustomEvent('subflow:maintenance', { detail: { state: 'migrating' } })))

  const dialog = page.getByRole('dialog')
  await expect(dialog).toBeVisible()
  await expect(dialog.getByRole('heading')).toBeFocused()
  await expect(email).toHaveValue('not-yet-saved@example.test')
  await expect(password).toHaveValue('unsaved-password')
  await expect(dialog).toHaveAttribute('open', '')

  phase = 'ready'
  ready = true
  await expect(dialog).toBeHidden({ timeout: 10000 })
  await expect(password).toBeFocused()
  await expect(password).toHaveValue('unsaved-password')
})
