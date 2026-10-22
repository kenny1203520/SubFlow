import { expect, test, type Page } from '@playwright/test'
import { randomUUID } from 'node:crypto'

async function signIn(page: Page, browserName: string) {
  const email = `share-${browserName}-${randomUUID()}@example.test`
  const registration = await page.request.post('/api/subflow/v1/auth/register', {
    data: { adminName: `Share ${browserName}`, email, password: 'test-password-1234' },
  })
  expect(registration.status(), await registration.text()).toBe(201)
  await page.goto('/auth')
  await page.getByLabel('Email').fill(email)
  await page.locator('input[autocomplete="current-password"]').fill('test-password-1234')
  await page.locator('form').getByRole('button', { name: 'Log in' }).click()
  await expect(page).toHaveURL(/personal/)
  await page.goto('/personal/share')
  await expect(page.getByRole('heading', { name: 'Share pages', exact: true })).toBeVisible()
  await page.getByRole('button', { name: 'Not now' }).click({ timeout: 1500 }).catch(() => {})
}

async function expectNoOverflow(page: Page, width: number) {
  const sizes = await page.evaluate(() => ({
    viewport: document.documentElement.clientWidth,
    document: document.documentElement.scrollWidth,
    body: document.body.scrollWidth,
    overflowing: [...document.querySelectorAll<HTMLElement>('body *')]
      .map(element => ({
        tag: element.tagName,
        className: typeof element.className === 'string' ? element.className : '',
        right: Math.round(element.getBoundingClientRect().right),
      }))
      .filter(element => element.right > document.documentElement.clientWidth + 1)
      .slice(0, 10),
  }))
  expect(sizes.viewport).toBe(width)
  expect(sizes.document, JSON.stringify(sizes)).toBeLessThanOrEqual(width + 1)
  expect(sizes.body, JSON.stringify(sizes)).toBeLessThanOrEqual(width + 1)
}

test('creates, edits, disables and opens a share, while fixed-date validation explains missing dates', async ({ page, browserName }) => {
  await page.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))
  await signIn(page, browserName)
  await page.getByRole('button', { name: /create share/i }).first().click()
  await page.getByRole('textbox', { name: 'Name', exact: true }).fill(`E2E ${browserName}`)
  await page.keyboard.press('Tab')
  await expect(page.locator('form')).toBeVisible()
  await page.getByText('Fixed dates', { exact: true }).click()
  await page.locator('[name="startsOn"]').fill('2026-08-31')
  await page.locator('form').getByRole('button', { name: 'Save settings' }).click()
  await expect(page.getByText('Choose an end date.')).toBeVisible()
  await expect(page.locator('[name="endsOn"]')).toBeFocused()
  await expect(page.getByRole('textbox', { name: 'Name', exact: true })).toHaveValue(`E2E ${browserName}`)

  await page.locator('[name="endsOn"]').fill('2026-08-31')
  await page.evaluate(() => Object.defineProperty(navigator, 'clipboard', {
    configurable: true,
    value: { writeText: async () => { throw new Error('clipboard permission denied') } },
  }))
  await page.locator('form').getByRole('button', { name: 'Save settings' }).click()
  await expect(page.getByRole('heading', { name: `E2E ${browserName}` })).toBeVisible()
  await expect(page.getByText(/Share created\. Copy this link manually:/)).toBeVisible()
  const item = page.locator('.share-item').filter({ has: page.getByRole('heading', { name: `E2E ${browserName}`, exact: true }) })
  const open = item.getByRole('button', { name: 'Open share page' })
  const [shareTab] = await Promise.all([page.context().waitForEvent('page'), open.click()])
  const publicResponse = await shareTab.waitForResponse(response => response.url().includes('/api/subflow/v1/shares/'))
  expect(publicResponse.status(), await publicResponse.text()).toBe(200)
  await expect(shareTab.getByRole('heading', { name: `E2E ${browserName}` })).toBeVisible()
  await shareTab.close()

  await item.getByRole('button', { name: 'Edit' }).click()
  await expect(page.locator('[name="startsOn"]')).toHaveValue('2026-08-31')
  await expect(page.locator('[name="endsOn"]')).toHaveValue('2026-08-31')
  await page.getByLabel('Enabled').uncheck()
  await page.locator('form').getByRole('button', { name: 'Save settings' }).click()
  await expect(item.locator('.status-pill')).toHaveText('Disabled')
})

for (const [timezoneId, width, colorScheme] of [
    ['Asia/Taipei', 375, 'dark'], ['UTC', 768, 'light'], ['America/New_York', 1024, 'dark'],
    ['Asia/Taipei', 1440, 'light'],
  ] as const) {
  test.describe(`date ranges ${timezoneId} at ${width}px`, () => {
    test.use({ timezoneId, viewport: { width, height: 900 }, screen: { width, height: 900 }, colorScheme })
    test('date ranges round-trip in supported timezones and narrow screens', async ({ page, browserName }) => {
    await page.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))
    await signIn(page, browserName)
    await page.getByRole('button', { name: /create share/i }).first().click()
    await page.getByText('Fixed dates', { exact: true }).click()
    await page.locator('[name="startsOn"]').fill('2026-08-31')
    await page.locator('[name="endsOn"]').fill('2026-08-31')
    const shareName = `Date check ${timezoneId} ${browserName}`
    await page.getByRole('textbox', { name: 'Name', exact: true }).fill(shareName)
    await page.evaluate(() => Object.defineProperty(navigator, 'clipboard', {
      configurable: true,
      value: { writeText: async () => { throw new Error('clipboard permission denied') } },
    }))
    const save = page.locator('form').getByRole('button', { name: 'Save settings' })
    await save.scrollIntoViewIfNeeded()
    await expectNoOverflow(page, width)
    await page.screenshot({ path: test.info().outputPath('share-editor.png'), fullPage: true })
    await save.click()
    await expect(page.locator('.share-item').getByRole('heading', { name: shareName, exact: true })).toBeVisible()
    await expect(page.getByText(/Share created\. Copy this link manually:/)).toBeVisible()
    const feedback = page.locator('.notice.success').first()
    expect(await feedback.evaluate(element => element.scrollWidth <= element.clientWidth + 1)).toBe(true)
    await expectNoOverflow(page, width)
    })
  })
}
