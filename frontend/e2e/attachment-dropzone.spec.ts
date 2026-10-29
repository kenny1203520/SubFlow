import { expect, test, type Page } from '@playwright/test'
import { randomUUID } from 'node:crypto'

async function signIn(page: Page, browserName: string) {
  const email = `attachment-${browserName}-${randomUUID()}@example.test`
  const registration = await page.request.post('/api/subflow/v1/auth/register', {
    data: { adminName: `Attachment ${browserName}`, email, password: 'test-password-1234' },
  })
  expect(registration.status(), await registration.text()).toBe(201)
  await page.goto('/auth')
  await page.getByLabel('Email').fill(email)
  await page.locator('input[autocomplete="current-password"]').fill('test-password-1234')
  await page.locator('form').getByRole('button', { name: 'Log in' }).click()
  await expect(page).toHaveURL(/home/)
}

test('adds files to a new expense draft on desktop and touch layouts without uploading early', async ({ page, browserName }, testInfo) => {
  await page.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))
  await signIn(page, browserName)
  await page.goto('/personal/expenses')
  await expect(page.getByRole('heading', { name: 'Personal expenses' })).toBeVisible()
  await page.getByRole('button', { name: 'Add expense' }).click()
  const dropzone = page.locator('.attachment-dropzone-wrap')
  await expect(dropzone).toBeVisible()

  const requests: string[] = []
  page.on('request', request => {
    if (request.url().includes('/attachments')) requests.push(request.method() + ' ' + request.url())
  })
  const files = [
    { name: 'receipt-a.pdf', mimeType: 'application/pdf', buffer: Buffer.from('%PDF-a') },
    { name: 'receipt-b.pdf', mimeType: 'application/pdf', buffer: Buffer.from('%PDF-b') },
  ]
  if (testInfo.project.name.includes('touch')) {
    await page.locator('input[type=file]').setInputFiles(files)
  } else {
    await dropzone.evaluate((element, fileSpecs) => {
      const data = new DataTransfer()
      for (const file of fileSpecs) data.items.add(new File([file.body], file.name, { type: file.type }))
      const enter = new DragEvent('dragenter', { bubbles: true, cancelable: true, dataTransfer: data })
      element.dispatchEvent(enter)
      element.dispatchEvent(new DragEvent('dragover', { bubbles: true, cancelable: true, dataTransfer: data }))
      element.dispatchEvent(new DragEvent('drop', { bubbles: true, cancelable: true, dataTransfer: data }))
    }, files.map(file => ({ name: file.name, type: file.mimeType, body: file.buffer.toString() })))
  }

  await expect(page.locator('.attachment-draft-list li')).toHaveCount(2)
  await expect(page.getByRole('status').filter({ hasText: '2 files selected' })).toBeVisible()
  await expect(page.getByText('receipt-a.pdf')).toBeVisible()
  await expect(page.getByText('receipt-b.pdf')).toBeVisible()
  expect(requests).toEqual([])
  const widths = await page.evaluate(() => ({ viewport: document.documentElement.clientWidth, document: document.documentElement.scrollWidth }))
  expect(widths.document).toBeLessThanOrEqual(widths.viewport + 1)

  await page.getByRole('button', { name: 'Cancel' }).click()
  await page.getByRole('button', { name: 'Add expense' }).click()
  await expect(page.locator('.attachment-draft-list')).toHaveCount(0)
})
