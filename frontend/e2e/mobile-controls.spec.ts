import { expect, test } from '@playwright/test'
import { account } from './fixtures/session'

test('mobile form controls use readable text and preserve browser zoom', async ({ page, request }, testInfo) => {
  await account(page, request)
  await page.goto('/personal')
  await page.getByRole('button', { name: 'Not now' }).click({ timeout: 1500 }).catch(() => {})
  await page.getByRole('button', { name: 'Add record', exact: true }).click()
  const drawer = page.getByRole('dialog', { name: 'Add record', exact: true })
  for (const width of [320, 390, 768, 1440]) {
    await page.setViewportSize({ width, height: 900 })
    if (width <= 700 || testInfo.project.use.hasTouch) {
      const fields = await drawer.locator('input:not([type="checkbox"]):not([type="radio"]):not([type="file"]):not([type="hidden"]), select, textarea').evaluateAll(elements => elements.filter(element => element.getClientRects().length).map(element => ({
        tag: element.tagName, type: element.getAttribute('type'), fontSize: parseFloat(getComputedStyle(element).fontSize),
      })))
      expect(fields.every(field => field.fontSize >= 16), JSON.stringify(fields)).toBe(true)
    }
  }
  const viewport = await page.locator('meta[name="viewport"]').getAttribute('content')
  expect(viewport).not.toMatch(/user-scalable\s*=\s*(no|0)|maximum-scale\s*=\s*1(?:[,\s]|$)/)
  expect(await page.evaluate(() => getComputedStyle(document.body).touchAction)).toBe('auto')
})
