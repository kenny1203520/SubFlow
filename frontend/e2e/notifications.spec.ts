import { expect, test, type Page } from '@playwright/test'
import { randomUUID } from 'node:crypto'

async function signIn(page: Page, name: string) {
  const email = `notifications-${name}-${randomUUID()}@example.test`
  const registration = await page.request.post('/api/subflow/v1/auth/register', {
    data: { adminName: `Notifications ${name}`, email, password: 'test-password-1234' },
  })
  expect(registration.status(), await registration.text()).toBe(201)
  await page.goto('/auth')
  await page.getByLabel('Email').fill(email)
  await page.locator('input[autocomplete="current-password"]').fill('test-password-1234')
  await page.locator('form').getByRole('button', { name: 'Log in' }).click()
  await expect(page).toHaveURL(/home/)
}

test('notification center preferences and bell fit supported widths and themes', async ({ browser, browserName }) => {
  const cases = [
    { width: 375, colorScheme: 'dark' },
    { width: 768, colorScheme: 'light' },
    { width: 1024, colorScheme: 'dark' },
    { width: 1440, colorScheme: 'light' },
  ] as const

  for (const [index, scenario] of cases.entries()) {
    const context = await browser.newContext({ viewport: { width: scenario.width, height: 900 }, colorScheme: scenario.colorScheme })
    const page = await context.newPage()
    await page.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))
    await signIn(page, `${browserName}-${index}`)
    await page.goto('/notifications')
    await expect(page.getByRole('heading', { name: 'Notification center' })).toBeVisible()

    const preferences = page.locator('#notification-preferences-title')
    await expect(preferences).toHaveAttribute('aria-expanded', 'false')
    await preferences.click()
    await expect(preferences).toHaveAttribute('aria-expanded', 'true')
    await expect(page.getByRole('table')).toBeVisible()
    await expect(page.getByRole('checkbox', { name: 'Security — In-app alerts' })).toBeDisabled()
    await expect(page.getByRole('checkbox', { name: 'Security — Email' })).toBeDisabled()
    await page.getByRole('button', { name: 'Save preferences' }).click()
    await expect(page.getByText('Notification preferences saved.')).toBeVisible()

    const dimensions = await page.evaluate(() => ({ viewport: window.innerWidth, document: document.documentElement.scrollWidth }))
    expect(dimensions.document, JSON.stringify(dimensions)).toBeLessThanOrEqual(dimensions.viewport)

    await page.getByRole('button', { name: 'Notifications' }).click()
    await expect(page.getByRole('dialog', { name: 'Notifications' })).toBeVisible()
    await expect(page.getByRole('dialog', { name: 'Notifications' }).getByText('You are all caught up')).toBeVisible()
    await page.keyboard.press('Escape')
    await context.close()
  }
})
