import { expect, test } from '@playwright/test'
import { account } from './fixtures/session'

test('overview metric values remain readable without clipping at supported widths', async ({ page, request }, testInfo) => {
  const { headers, userID } = await account(page, request)
  expect((await request.post('/api/subflow/v1/expenses', { headers, data: {
    title: 'Long financial value', amountMinor: 123456789012345, currency: 'TWD', incurredOn: new Date().toISOString(), paidBy: userID,
  } })).status()).toBe(201)
  await page.goto('/overview?scope=personal')
  await expect(page.locator('.metric-strip article').first()).toBeVisible()
  for (const width of [320, 375, 390, 768, 1024, 1440]) {
    await page.setViewportSize({ width, height: 900 })
    const cards = await page.locator('.metric-strip article').evaluateAll(elements => elements.map(element => {
      const parent = element.parentElement!.getBoundingClientRect(), rect = element.getBoundingClientRect()
      return {
        width: rect.width,
        clipped: element.scrollWidth > element.clientWidth + 1,
        childClipped: [...element.children].some(child => child.scrollWidth > child.clientWidth + 1),
        outside: rect.right > parent.right + 1 || rect.left < parent.left - 1,
      }
    }))
    expect(cards.every(card => !card.clipped && !card.childClipped && !card.outside), JSON.stringify({ width, cards })).toBe(true)
    expect(Math.max(...cards.map(card => card.width)) - Math.min(...cards.map(card => card.width))).toBeLessThanOrEqual(1)
    if (width === 320 || width === 1440) await page.screenshot({ path: testInfo.outputPath(`overview-${width}.png`), fullPage: true })
  }
})
