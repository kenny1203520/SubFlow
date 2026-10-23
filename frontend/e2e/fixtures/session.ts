import { expect, type APIRequestContext, type Page } from "@playwright/test"
import { randomUUID } from "node:crypto"

export async function account(page: Page, request: APIRequestContext) {
  const email = `mobile-forms-${randomUUID()}@example.test`, password = 'test-password-1234'
  expect((await request.post('/api/subflow/v1/auth/register', { data: { adminName: 'Mobile forms', email, password } })).status()).toBe(201)
  const response = await request.post('/api/collections/users/auth-with-password', { data: { identity: email, password } })
  expect(response.status()).toBe(200)
  const auth = await response.json()
  await page.addInitScript(value => {
    localStorage.setItem('subflow-locale', 'en')
    localStorage.setItem('pocketbase_auth', JSON.stringify({ token: value.token, record: value.record }))
  }, auth)
  return { headers: { Authorization: `Bearer ${auth.token}` }, userID: auth.record.id as string }
}

