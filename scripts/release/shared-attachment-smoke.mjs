import assert from 'node:assert/strict'
import { randomUUID } from 'node:crypto'

function pdfFixture() {
  const content = 'BT /F1 24 Tf 60 700 Td (Production PDF preview) Tj ET'
  const objects = [
    '<< /Type /Catalog /Pages 2 0 R >>',
    '<< /Type /Pages /Kids [3 0 R] /Count 1 >>',
    '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>',
    '<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>',
    `<< /Length ${content.length} >>\nstream\n${content}\nendstream`,
  ]
  let source = '%PDF-1.4\n'
  const offsets = []
  for (const [index, object] of objects.entries()) {
    offsets.push(Buffer.byteLength(source))
    source += `${index + 1} 0 obj\n${object}\nendobj\n`
  }
  const xref = Buffer.byteLength(source)
  source += `xref\n0 6\n0000000000 65535 f \n${offsets.map(offset => `${String(offset).padStart(10, '0')} 00000 n \n`).join('')}trailer\n<< /Size 6 /Root 1 0 R >>\nstartxref\n${xref}\n%%EOF`
  return Buffer.from(source)
}

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
      file: { name: 'receipt.pdf', mimeType: 'application/pdf', buffer: pdfFixture() },
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
      await expect(dialog.getByText('receipt.pdf', { exact: true })).toBeVisible()
      await dialog.getByRole('button', { name: 'Preview', exact: true }).click()
      await expect(dialog.locator('.pdf-preview canvas')).toBeVisible()
      await expect.poll(() => dialog.locator('canvas').evaluate(canvas => {
        const pixels = canvas.getContext('2d').getImageData(0, 0, canvas.width, canvas.height).data
        let dark = 0
        for (let i = 0; i < pixels.length; i += 4) if (pixels[i] < 80 && pixels[i + 1] < 80 && pixels[i + 2] < 80 && pixels[i + 3]) dark++
        return dark
      })).toBeGreaterThan(20)
      await page.keyboard.press('Escape')
      await expect(dialog).toBeHidden()
      await expect(trigger).toBeFocused()
    }
  } finally { await context.close() }
}
