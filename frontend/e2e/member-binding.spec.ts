import { expect, test } from '@playwright/test'
import { account } from './fixtures/session'

test('invite and bind brings the invitation form into view without sending an invitation', async ({ page, request }) => {
  const { headers } = await account(page, request)
  const response = await request.post('/api/subflow/v1/groups', { headers, data: { name: 'Binding UX', currency: 'TWD', timezone: 'Asia/Taipei' } })
  expect(response.status()).toBe(201)
  const group = (await response.json()).data
  expect((await request.post(`/api/subflow/v1/groups/${group.id}/temp-members`, { headers, data: { name: 'Pending participant' } })).status()).toBe(201)
  let sent = 0
  page.on('request', value => { if (value.method() === 'POST' && value.url().endsWith('/invitations')) sent++ })
  await page.goto(`/groups/${group.id}/members`)
  for (const width of [320, 390, 768, 1440]) {
    await page.setViewportSize({ width, height: 844 })
    for (const reducedMotion of ['reduce', 'no-preference'] as const) {
      await page.emulateMedia({ reducedMotion })
      await page.getByRole('button', { name: 'Invite & bind', exact: true }).click()
      const form = page.locator('form').filter({ has: page.getByRole('heading', { name: 'Invite a member', exact: true }) })
      await expect(form.getByLabel('Email', { exact: true })).toBeFocused()
      await expect(form).toBeInViewport({ ratio: 0.9 })
      await expect(form.getByText('This invitation will bind to “Pending participant” once accepted.')).toBeVisible()
      await form.getByRole('button', { name: 'Cancel', exact: true }).click()
    }
  }
  expect(sent).toBe(0)
})
