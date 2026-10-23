import { expect, test, type Page } from '@playwright/test'

import { onePagePdf } from './fixtures/pdf'

async function expectRenderedPdf(page: Page) {
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
}

test('renders real PDF page pixels in the app preview instead of a browser PDF iframe', async ({ page }) => {
  await expectRenderedPdf(page)
})

test('renders PDF pages when Promise.try is unavailable', async ({ page }) => {
  // Capability injection models API support differences; not physical iOS.
  await page.addInitScript(() => {
    Object.defineProperty(Promise, 'try', { configurable: true, writable: true, value: undefined })
  })
  let workerRequested = false
  await page.context().route('**/*pdf.worker*.mjs*', async route => {
    workerRequested = true
    const response = await route.fetch()
    await route.fulfill({ response, body: 'Promise.try = undefined;\n' + await response.text() })
  })
  await expectRenderedPdf(page)
  expect(workerRequested).toBe(true)
})
