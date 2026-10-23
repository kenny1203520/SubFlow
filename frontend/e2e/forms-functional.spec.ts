import { expect, test, type Locator } from '@playwright/test'
import { account } from './fixtures/session'
import { onePagePdf } from './fixtures/pdf'
import { randomUUID } from 'node:crypto'

async function expectPdfContent(canvas: Locator) {
  await expect(canvas).toBeVisible()
  await expect.poll(() => canvas.evaluate((element: HTMLCanvasElement) => {
    const context = element.getContext('2d')
    if (!context || !element.width || !element.height) return 0
    const pixels = context.getImageData(0, 0, element.width, element.height).data
    let dark = 0
    for (let index = 0; index < pixels.length; index += 4) {
      if (pixels[index] < 80 && pixels[index + 1] < 80 && pixels[index + 2] < 80 && pixels[index + 3] > 0) dark++
    }
    return dark
  })).toBeGreaterThan(20)
}

test('financial drawers and profile selectors accept pointer and keyboard input without saving on cancel', async ({ page, request }, testInfo) => {
  test.setTimeout(180_000)
  const { headers } = await account(page, request)
  const createdGroup = await request.post('/api/subflow/v1/groups', { headers, data: { name: 'Form control coverage', currency: 'TWD', timezone: 'Asia/Taipei' } })
  expect(createdGroup.status()).toBe(201)
  const groupID = (await createdGroup.json()).data.id as string
  let writes = 0
  page.on('request', value => { if (['POST', 'PATCH', 'DELETE'].includes(value.method()) && /\/api\/subflow\/v1\/(expenses|incomes|subscriptions)(\/|$)/.test(new URL(value.url()).pathname)) writes++ })
  for (const width of [320, 768, 1440]) {
    await page.setViewportSize({ width, height: 900 })
    for (const [path, title] of [
      ['/personal/expenses', 'Add expense'], ['/personal/incomes', 'Add income'], ['/personal/subscriptions', 'Add subscription'],
      [`/groups/${groupID}/expenses`, 'Add expense'], [`/groups/${groupID}/incomes`, 'Add income'], [`/groups/${groupID}/subscriptions`, 'Add subscription'],
    ]) {
      await page.goto(path!)
      await page.getByRole('button', { name: 'Not now' }).click({ timeout: 500 }).catch(() => {})
      await page.getByRole('button', { name: title, exact: true }).click()
      const drawer = page.getByRole('dialog', { name: title, exact: true })
      if (width === 320 || width === 1440) await page.screenshot({ path: testInfo.outputPath(`${title!.replaceAll(' ', '-')}-${width}.png`) })
      await drawer.locator('.category-trigger').click()
      const category = page.locator('.category-menu')
      await category.getByRole('button', { name: 'Uncategorized', exact: true }).click()
      await expect(category).toBeHidden()
      if (await drawer.locator('.currency-trigger').count()) {
        await drawer.locator('.currency-trigger').click()
        const currencies = page.locator('.currency-dropdown')
        await currencies.locator('input').fill('TWD')
        await currencies.locator('.currency-option').first().click()
        await expect(currencies).toBeHidden()
      }
      if (title === 'Add subscription') {
        await drawer.getByLabel('Cycle', { exact: true }).click()
        const cycle = page.locator('.combobox-menu')
        await cycle.getByRole('button', { name: 'Monthly', exact: true }).click()
        await expect(cycle).toBeHidden()
      }
      const fields = drawer.locator('input:not([type="hidden"]):not([type="file"]):not([type="checkbox"]):not([type="radio"]), textarea')
      for (const field of await fields.all()) {
        if (!await field.isVisible() || !await field.isEnabled()) continue
        await field.click()
        await expect(field).toBeFocused()
      }
      await drawer.getByRole('button', { name: 'Cancel', exact: true }).click()
      await expect(drawer).toBeHidden()
      expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth + 1)).toBe(true)
    }
  }
  expect(writes).toBe(0)
  await page.goto('/profile')
  await page.locator('.currency-trigger').first().click()
  await page.locator('.currency-dropdown input').fill('TWD')
  await page.locator('.currency-option').first().click()
  await expect(page.locator('.currency-dropdown')).toBeHidden()
  await page.locator('.timezone-trigger').first().click()
  await page.locator('.timezone-dropdown input').fill('Taipei')
  await page.locator('.timezone-option').first().click()
  await expect(page.locator('.timezone-dropdown')).toBeHidden()
})

test('an uploaded PDF renders through personal and shared attachment dialogs', async ({ page, request }, testInfo) => {
  const { headers, userID } = await account(page, request)
  const expense = await request.post('/api/subflow/v1/expenses', { headers, data: {
    title: 'PDF attachment path', amountMinor: 1200, currency: 'TWD', incurredOn: new Date().toISOString(), paidBy: userID,
  } })
  expect(expense.status()).toBe(201)
  const record = (await expense.json()).data
  expect((await request.post('/api/subflow/v1/attachments', { headers, multipart: {
    targetType: 'expense', targetId: record.id, requestId: randomUUID(),
    file: { name: 'real-receipt.pdf', mimeType: 'application/pdf', buffer: Buffer.from(onePagePdf()) },
  } })).status()).toBe(201)
  const errors: string[] = []
  page.on('pageerror', error => errors.push(error.name))
  await page.goto('/personal/expenses')
  await page.getByRole('button', { name: 'Not now' }).click({ timeout: 500 }).catch(() => {})
  await page.getByRole('button', { name: 'Attachments', exact: true }).click()
  await page.getByRole('button', { name: 'real-receipt.pdf', exact: true }).click()
  await expectPdfContent(page.locator('.attachment-preview .pdf-preview canvas'))
  await expect(page.locator('.attachment-preview [role="alert"]')).toHaveCount(0)
  await page.getByRole('dialog', { name: 'Attachment preview', exact: true }).getByRole('button', { name: 'Close', exact: true }).click()
  await page.getByRole('dialog', { name: 'Attachments', exact: true }).getByRole('button', { name: 'Close', exact: true }).click()
  const share = await request.post('/api/subflow/v1/personal/shares', { headers, data: {
    name: 'Shared PDF', accessMode: 'link', enabled: true, rangeMode: 'all', showExpenses: true, showAttachments: true,
  } })
  expect(share.status()).toBe(201)
  await page.goto((await share.json()).data.url)
  await page.locator('.share-attachment-trigger').click()
  const dialog = page.getByRole('dialog', { name: 'Attachments', exact: true })
  await dialog.getByRole('button', { name: 'Preview', exact: true }).click()
  await expectPdfContent(dialog.locator('.pdf-preview canvas'))
  await page.screenshot({ path: testInfo.outputPath('shared-pdf-preview.png') })
  await expect(dialog.getByRole('alert')).toHaveCount(0)
  await page.keyboard.press('Escape')
  await expect(dialog).toBeHidden()
  expect(errors).toEqual([])
})
