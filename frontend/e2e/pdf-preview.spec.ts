import { expect, test } from '@playwright/test'

function onePagePdf(): Uint8Array {
  const content = 'BT /F1 24 Tf 60 700 Td (PDF preview renders text) Tj ET'
  const objects = [
    '<< /Type /Catalog /Pages 2 0 R >>',
    '<< /Type /Pages /Kids [3 0 R] /Count 1 >>',
    '<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 4 0 R >> >> /Contents 5 0 R >>',
    '<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>',
    `<< /Length ${content.length} >>\nstream\n${content}\nendstream`,
  ]
  let source = '%PDF-1.4\n'
  const offsets = [0]
  for (let index = 0; index < objects.length; index++) {
    offsets.push(new TextEncoder().encode(source).length)
    source += `${index + 1} 0 obj\n${objects[index]}\nendobj\n`
  }
  const xref = new TextEncoder().encode(source).length
  source += `xref\n0 ${objects.length + 1}\n0000000000 65535 f \n${offsets.slice(1).map(offset => `${String(offset).padStart(10, '0')} 00000 n \n`).join('')}trailer\n<< /Size ${objects.length + 1} /Root 1 0 R >>\nstartxref\n${xref}\n%%EOF`
  return new TextEncoder().encode(source)
}

test('renders real PDF page pixels in the app preview instead of a browser PDF iframe', async ({ page }) => {
  await page.goto('/auth')
  const bytes = [...onePagePdf()]
  await page.evaluate(async (pdfBytes) => {
    const { mountPdfPreview } = await import('/e2e/pdf-preview-harness.ts')
    const url = URL.createObjectURL(new Blob([new Uint8Array(pdfBytes)], { type: 'application/pdf' }))
    mountPdfPreview(url)
  }, bytes)
  const canvas = page.locator('.pdf-preview canvas')
  await expect(canvas).toBeVisible()
  await expect.poll(() => canvas.evaluate((element) => {
    const canvasElement = element as HTMLCanvasElement
    const context = canvasElement.getContext('2d')
    if (!context || !canvasElement.width || !canvasElement.height) return 0
    const pixels = context.getImageData(0, 0, canvasElement.width, canvasElement.height).data
    let dark = 0
    for (let index = 0; index < pixels.length; index += 4) {
      if (pixels[index] < 80 && pixels[index + 1] < 80 && pixels[index + 2] < 80 && pixels[index + 3] > 0) dark++
    }
    return dark
  })).toBeGreaterThan(20)
  await expect(page.locator('iframe')).toHaveCount(0)
})
