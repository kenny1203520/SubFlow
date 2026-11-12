import { expect, test } from '@playwright/test'

const providers = [
  { name: 'authentik', displayName: 'Company identity with a long provider name', logo: '' },
  { name: 'spotify', displayName: 'Spotify', logo: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"><circle /></svg>' },
  { name: 'google', displayName: 'Google', logo: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"><circle /></svg>' },
  { name: 'github', displayName: 'GitHub', logo: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"><circle /></svg>' },
  { name: 'discord', displayName: 'Discord', logo: '<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1 1"><circle /></svg>' },
]

const authMethods = (oauthProviders: typeof providers) => ({
  mfa: { enabled: false, duration: 0 },
  otp: { enabled: true, duration: 300 },
  password: { enabled: true, identityFields: ['email'] },
  oauth2: { enabled: oauthProviders.length > 0, providers: oauthProviders },
})

test('OAuth provider options stay readable, branded and keyboard-ordered at narrow widths', async ({ page }) => {
  test.skip(test.info().project.name !== 'chromium', 'Responsive acceptance runs on Chromium.')
  await page.addInitScript(() => {
    if (!localStorage.getItem('subflow-locale')) localStorage.setItem('subflow-locale', 'en')
  })
  let availableProviders: typeof providers = []
  await page.route('**/auth-methods**', route => route.fulfill({
    status: 200,
    contentType: 'application/json',
    body: JSON.stringify(authMethods(availableProviders)),
  }))

  await page.goto('/auth')
  await expect(page.getByRole('button', { name: 'Sign in with an email code' })).toBeVisible()
  await expect(page.getByRole('group', { name: 'Or sign in with a provider' })).toHaveCount(0)

  availableProviders = providers
  await page.reload()
  const group = page.getByRole('group', { name: 'Or sign in with a provider' })
  await expect(group).toBeVisible()
  await expect(group.getByRole('button')).toHaveCount(5)
  await expect(group.getByRole('button', { name: 'Continue with Google' })).toBeVisible()
  await expect(group).toContainText('Company identity with a long provider name')

  for (const width of [320, 375, 1440]) {
    await page.setViewportSize({ width, height: 900 })
    const layout = await group.locator('button').evaluateAll(buttons => ({
      viewport: document.documentElement.clientWidth,
      document: document.documentElement.scrollWidth,
      targets: buttons.map(button => {
        const rect = button.getBoundingClientRect()
        return { top: Math.round(rect.top), width: rect.width, height: rect.height }
      }),
    }))
    expect(layout.document, `horizontal overflow at ${width}px`).toBeLessThanOrEqual(layout.viewport + 1)
    expect(layout.targets.every(target => target.width > 0 && target.height >= 48), `target size at ${width}px`).toBe(true)
    if (width === 1440) expect(new Set(layout.targets.map(target => target.top)).size).toBe(3)
  }

  const buttons = group.getByRole('button')
  await buttons.nth(0).focus()
  for (let index = 0; index < 4; index++) {
    await page.keyboard.press('Tab')
    await expect(buttons.nth(index + 1)).toBeFocused()
  }

  await page.locator('html').evaluate(element => element.setAttribute('data-theme', 'light'))
  await expect(group.getByRole('button', { name: 'Continue with Google' })).toHaveCSS('background-color', 'rgb(255, 255, 255)')
  await page.locator('html').evaluate(element => element.setAttribute('data-theme', 'dark'))
  await expect(group.getByRole('button', { name: 'Continue with Google' })).toHaveCSS('background-color', 'rgb(19, 19, 20)')

  await page.evaluate(() => localStorage.setItem('subflow-locale', 'zh-TW'))
  await page.reload()
  await expect(page.getByRole('group', { name: '或使用第三方帳號登入' })).toBeVisible()
  await expect(page.getByRole('button', { name: '使用 Google 登入' })).toBeVisible()

  await page.evaluate(() => localStorage.setItem('subflow-locale', 'en'))
  await page.reload()
  await page.getByLabel('Email', { exact: true }).fill('owner@example.test')
  await page.locator('input[autocomplete="current-password"]').fill('test-password-1234')
  await page.locator('form').getByRole('button', { name: 'Log in', exact: true }).click()
  await expect(page).toHaveURL(/home/)
  await page.goto('/profile')

  const profileProviders = page.locator('.profile-providers')
  const longProviderName = profileProviders.locator('.profile-provider-name strong').filter({
    hasText: 'Company identity with a long provider name',
  })
  await expect(profileProviders).toBeVisible()
  await expect(longProviderName).toBeVisible()
  expect(await longProviderName.evaluate(element => getComputedStyle(element).whiteSpace)).toBe('normal')
  expect(await longProviderName.evaluate(element => getComputedStyle(element).overflow)).toBe('visible')
})
