import { expect, test } from '@playwright/test'
import { randomUUID } from 'node:crypto'

test('a share with an attachment keeps its closed attachment dialog hidden', async ({ page }) => {
  await page.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))
  // A minimal read-only projection isolates rendering from backend mutations.
  await page.route('**/api/subflow/v1/shares/render-check?**', route => route.fulfill({ json: { data: {
    name: 'Shared ledger with receipt', currency: 'TWD', rangeLabel: 'all', showSummary: false,
    records: [{ type: 'expense', title: 'Receipt expense', amountMinor: 12300, currency: 'TWD', occurredOn: '2026-10-23T00:00:00Z', attachmentReference: 'receipt', attachmentCount: 1 }],
    totalItems: 1, totalPages: 1, page: 1, perPage: 25,
  } } }))
  await page.setViewportSize({ width: 390, height: 844 })
  await page.goto('/share/render-check')
  await expect(page.getByRole('heading', { name: 'Shared ledger with receipt' })).toBeVisible()
  const closedDialog = page.locator('dialog.share-attachment-dialog:not([open])')
  await expect(closedDialog).toHaveCount(1)
  await page.screenshot({ path: test.info().outputPath('share-attachment-closed.png'), fullPage: true })
  await expect(closedDialog).toBeHidden()
  await page.getByRole('button', { name: 'Newest first' }).click()
})

for (const mode of ['link', 'password', 'accounts'] as const) {
  test(`real ${mode} share attachments remain usable at every supported width`, async ({ page, request, browser }, testInfo) => {
    const runtimeErrors: string[] = []
    page.on('pageerror', error => runtimeErrors.push(error.name))
    const email = `share-view-${randomUUID()}@example.test`
    const password = 'test-password-1234'
    const registered = await request.post('/api/subflow/v1/auth/register', { data: { adminName: 'Share view regression', email, password } })
    expect(registered.status()).toBe(201)
    const authenticated = await request.post('/api/collections/users/auth-with-password', { data: { identity: email, password } })
    expect(authenticated.status()).toBe(200)
    const auth = await authenticated.json()
    const headers = { Authorization: `Bearer ${auth.token}` }
    const expense = await request.post('/api/subflow/v1/expenses', { headers, data: {
      title: 'Visible shared receipt', amountMinor: 12300, currency: 'TWD', incurredOn: '2026-10-23T00:00:00Z', paidBy: auth.record.id,
    } })
    expect(expense.status()).toBe(201)
    const record = (await expense.json()).data
    const uploaded = await request.post('/api/subflow/v1/attachments', { headers, multipart: {
      targetType: 'expense', targetId: record.id, requestId: randomUUID(),
      file: { name: 'receipt.png', mimeType: 'image/png', buffer: Buffer.from('iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAQAAAC1HAwCAAAAC0lEQVR42mP8/x8AAwMCAO+aGZkAAAAASUVORK5CYII=', 'base64') },
    } })
    expect(uploaded.status()).toBe(201)
    const shared = await request.post('/api/subflow/v1/personal/shares', { headers, data: {
      name: `Receipt share ${mode}`, accessMode: mode, password: mode === 'password' ? password : '', viewerEmails: mode === 'accounts' ? [email] : [],
      enabled: true, rangeMode: 'all', showExpenses: true, showSummary: true, showAttachments: true,
    } })
    expect(shared.status()).toBe(201)
    const sharePath = (await shared.json()).data.url as string
    await page.addInitScript(({ account, savedAuth }) => {
      localStorage.setItem('subflow-locale', 'en')
      if (account) localStorage.setItem('pocketbase_auth', JSON.stringify({ token: savedAuth.token, record: savedAuth.record }))
    }, { account: mode === 'accounts', savedAuth: auth })
    await page.goto(sharePath)
    if (mode === 'password') {
      await expect(page.locator('.unlock-card')).toBeVisible()
      await page.getByLabel('Password', { exact: true }).fill(password)
      await page.locator('.unlock-card button').click()
    }
    await expect(page.getByRole('heading', { name: `Receipt share ${mode}` })).toBeVisible()
    await expect(page.getByText('Visible shared receipt', { exact: true })).toBeVisible()
    const trigger = page.locator('.share-attachment-trigger')
    const dialog = page.locator('dialog.share-attachment-dialog')
    await expect(trigger).toHaveCount(1)

    for (const width of [320, 375, 390, 768, 1024, 1440]) {
      await page.setViewportSize({ width, height: 900 })
      for (const theme of ['light', 'dark']) {
        await page.evaluate(value => document.documentElement.dataset.theme = value, theme)
        await expect(dialog).toBeHidden()
        const sizes = await page.evaluate(() => ({ document: document.documentElement.scrollWidth, viewport: document.documentElement.clientWidth }))
        expect(sizes.viewport).toBe(width)
        expect(sizes.document).toBeLessThanOrEqual(width + 1)
        await trigger.click()
        await expect(dialog).toBeVisible()
        await expect(dialog.getByText('receipt.png', { exact: true })).toBeVisible()
        const bounds = await dialog.boundingBox()
        expect(bounds).not.toBeNull()
        expect(bounds!.x).toBeGreaterThanOrEqual(-1)
        expect(bounds!.x + bounds!.width).toBeLessThanOrEqual(width + 1)
        await page.keyboard.press('Escape')
        await expect(dialog).toBeHidden()
        await expect(trigger).toBeFocused()
      }
      if (width === 320 || width === 1440) await page.screenshot({ path: testInfo.outputPath(`share-${mode}-${width}.png`), fullPage: true })
    }

    // A rejected attachment read is injected; reopening retries the real API.
    await page.route('**/attachments/**', route => route.fulfill({ status: 503, json: { error: { code: 'temporarily_unavailable' } } }), { times: 1 })
    await trigger.click()
    await expect(dialog.getByRole('alert')).toBeVisible()
    await dialog.getByRole('button', { name: 'Close', exact: true }).click()
    await expect(dialog).toBeHidden()
    await trigger.click()
    await expect(dialog.getByRole('alert')).toHaveCount(0)
    await dialog.getByRole('button', { name: 'Preview', exact: true }).click()
    await expect(dialog.locator('img')).toBeVisible()
    await expect.poll(() => dialog.locator('img').evaluate(element => (element as HTMLImageElement).naturalWidth)).toBeGreaterThan(0)
    await page.keyboard.press('Escape')
    await expect(dialog).toBeHidden()
    await trigger.click()
    const download = page.waitForEvent('download')
    await dialog.getByRole('button', { name: 'Download', exact: true }).click()
    expect((await download).suggestedFilename()).toBe('receipt.png')
    await dialog.getByRole('button', { name: 'Close', exact: true }).click()
    await expect(dialog).toBeHidden()
    expect(runtimeErrors).toEqual([])

    if (mode !== 'link') {
      const visitor = await browser.newContext()
      try {
        const outsider = await visitor.newPage()
        await outsider.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))
        await outsider.goto(sharePath)
        if (mode === 'password') {
          await expect(outsider.locator('.unlock-card')).toBeVisible()
          await outsider.getByLabel('Password', { exact: true }).fill('wrong-password')
          await outsider.locator('.unlock-card button').click()
          await expect(outsider.locator('.error-state')).toBeVisible()
        } else await expect(outsider.locator('input[autocomplete="current-password"]')).toBeVisible()
        await expect(outsider.locator('.share-attachment-trigger')).toHaveCount(0)
        await expect(outsider.locator('dialog[open]')).toHaveCount(0)
      } finally { await visitor.close() }
    }
  })
}
