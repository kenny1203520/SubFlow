import assert from 'node:assert/strict'
import { randomUUID } from 'node:crypto'

// Exercise the bundled production CSS and a real anonymous attachment share.
export async function verifySharedAttachment(browser, baseURL, token, ownerID, expect) {
  const context = await browser.newContext({ baseURL, viewport: { width: 390, height: 844 }, hasTouch: true })
  try {
    const headers = { Authorization: `Bearer ${token}` }
    const expense = await context.request.post('/api/subflow/v1/expenses', { headers, data: {
      title: 'Production shared receipt', amountMinor: 12300, currency: 'TWD', incurredOn: '2026-10-23T00:00:00Z', paidBy: ownerID,
    } })
    assert.equal(expense.status(), 201)
    const record = (await expense.json()).data
    const uploaded = await context.request.post('/api/subflow/v1/attachments', { headers, multipart: {
      targetType: 'expense', targetId: record.id, requestId: randomUUID(),
      file: { name: 'receipt.png', mimeType: 'image/png', buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aGZkAAAAASUVORK5CYII=', 'base64') },
    } })
    assert.equal(uploaded.status(), 201)
    const shared = await context.request.post('/api/subflow/v1/personal/shares', { headers, data: {
      name: 'Production attachment share', accessMode: 'link', enabled: true, rangeMode: 'all', showExpenses: true, showAttachments: true,
    } })
    assert.equal(shared.status(), 201)
    const sharePath = (await shared.json()).data.url
    const page = await context.newPage()
    await page.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))
    await page.goto(sharePath)
    const dialog = page.locator('dialog.share-attachment-dialog')
    const trigger = page.locator('.share-attachment-trigger')
    await expect(trigger).toBeVisible()
    await expect(dialog).toHaveCount(1)
    await page.screenshot({ path: '.release/shared-attachment-mobile.png', fullPage: true })
    await expect(dialog).toBeHidden()
    for (const width of [390, 1440]) {
      await page.setViewportSize({ width, height: 900 })
      await trigger.click()
      await expect(dialog.getByText('receipt.png', { exact: true })).toBeVisible()
      await dialog.getByRole('button', { name: 'Preview', exact: true }).click()
      await expect.poll(() => dialog.locator('img').evaluate(element => element.naturalWidth)).toBeGreaterThan(0)
      await page.keyboard.press('Escape')
      await expect(dialog).toBeHidden()
      await expect(trigger).toBeFocused()
    }
  } finally { await context.close() }
}
