import { expect, test, type Page } from '@playwright/test'
import { randomUUID } from 'node:crypto'

async function signIn(page: Page, browserName: string) {
  const email = `share-attachments-${browserName}-${randomUUID()}@example.test`
  const response = await page.request.post('/api/subflow/v1/auth/register', {
    data: { adminName: 'Attachment E2E', email, password: 'test-password-1234' },
  })
  expect(response.status(), await response.text()).toBe(201)
  await page.goto('/auth')
  await page.getByLabel('Email').fill(email)
  await page.locator('input[autocomplete="current-password"]').fill('test-password-1234')
  await page.locator('form').getByRole('button', { name: 'Log in' }).click()
  await expect(page).toHaveURL(/home/)
  await page.goto('/personal/share')
  await page.getByRole('button', { name: 'Not now' }).click({ timeout: 1500 }).catch(() => {})
}

test('attachment sharing is opt-in and persists when editing a share', async ({ page, browserName }) => {
  await page.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))
  await signIn(page, browserName)
  const name = `Attachment share ${browserName}`
  await page.getByRole('button', { name: /create share/i }).first().click()
  await expect(page.getByLabel('Allow attachment viewing and downloads')).not.toBeChecked()
  await page.getByRole('textbox', { name: 'Name', exact: true }).fill(name)
  await page.getByLabel('Allow attachment viewing and downloads').check()
  await page.locator('form').getByRole('button', { name: 'Save settings' }).click()
  const item = page.locator('.share-item').filter({ has: page.getByRole('heading', { name, exact: true }) })
  await expect(item).toBeVisible()
  await item.getByRole('button', { name: 'Edit' }).click()
  await expect(page.getByLabel('Allow attachment viewing and downloads')).toBeChecked()
  await page.getByRole('button', { name: 'Cancel' }).click()
})
