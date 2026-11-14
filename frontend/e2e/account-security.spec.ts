import { randomUUID } from "node:crypto"
import { expect, test, type APIRequestContext, type Page } from "@playwright/test"

type AuthResponse = { token: string; record: Record<string, unknown> }

async function prepareAuthPage(page: Page, auth?: AuthResponse) {
  await page.addInitScript(value => {
    localStorage.setItem("subflow-locale", "en")
    if (value.auth) localStorage.setItem("pocketbase_auth", JSON.stringify(value.auth))
  }, { auth: auth ?? null })
  await page.route("**/api/subflow/v1/setup/status*", route => route.fulfill({
    status: 200,
    contentType: "application/json",
    body: JSON.stringify({ data: {
      initialized: true,
      allowRegistration: false,
      allowPasswordRegistration: false,
      passwordResetAvailable: true,
      captchaFlows: {},
    } }),
  }))
  await page.route("**/api/collections/users/auth-methods*", route => route.fulfill({
    status: 200,
    contentType: "application/json",
    body: JSON.stringify({
      password: { enabled: true },
      otp: { enabled: true },
      mfa: { enabled: true },
      oauth2: { enabled: false, providers: [] },
    }),
  }))
}

async function registerAndAuthenticate(request: APIRequestContext): Promise<{ email: string; password: string; auth: AuthResponse }> {
  const email = "security-" + randomUUID() + "@example.test"
  const password = "e2e-local-password-123"
  const created = await request.post("/api/subflow/v1/auth/register", {
    data: { adminName: "Account security test", email, password },
  })
  expect(created.status()).toBe(201)
  const response = await request.post("/api/collections/users/auth-with-password", {
    data: { identity: email, password },
  })
  expect(response.status()).toBe(200)
  return { email, password, auth: await response.json() as AuthResponse }
}

test("forgot-password is keyboard operable and fits a narrow viewport", async ({ page }) => {
  await prepareAuthPage(page)
  await page.setViewportSize({ width: 320, height: 760 })
  const submitted: Array<Record<string, unknown>> = []
  await page.route("**/api/collections/users/request-password-reset", async route => {
    submitted.push(route.request().postDataJSON() as Record<string, unknown>)
    await route.fulfill({ status: 204 })
  })

  await page.goto("/auth")
  await page.getByRole("button", { name: "Forgot password?" }).click()
  await expect(page.getByRole("heading", { name: "Forgot password?" })).toBeVisible()
  const email = page.getByLabel("Email")
  await expect(email).toBeFocused()
  await email.fill("person@example.test")
  await email.press("Tab")
  const submit = page.getByRole("button", { name: "Send verification code" })
  await expect(submit).toBeFocused()
  await page.keyboard.press("Enter")

  await expect(page.getByRole("status")).toContainText("A password-reset email has been sent")
  expect(submitted).toEqual([{ email: "person@example.test" }])
  expect(await page.evaluate(() => document.documentElement.scrollWidth <= document.documentElement.clientWidth + 1)).toBe(true)
})

test("reset links keep their token out of history and submit it only in the request body", async ({ page }) => {
  await prepareAuthPage(page)
  const token = "reset-link-token-for-e2e"
  let submitted: Record<string, unknown> | undefined
  let requestURL = ""
  await page.route("**/api/collections/users/confirm-password-reset", async route => {
    submitted = route.request().postDataJSON() as Record<string, unknown>
    requestURL = route.request().url()
    await route.fulfill({ status: 204 })
  })

  await page.goto("/auth/reset-password#" + token)
  await expect(page.locator("input[autocomplete=new-password]").first()).toBeVisible()
  await expect.poll(() => page.evaluate(() => window.location.hash)).toBe("")
  expect(page.url()).not.toContain(token)
  await page.locator("input[autocomplete=new-password]").first().fill("new-local-password-123")
  await page.locator("input[autocomplete=new-password]").nth(1).fill("new-local-password-123")
  await page.getByRole("button", { name: "Update password" }).click()

  await expect(page).toHaveURL(/\/auth\?notice=password-reset/)
  await expect(page.getByRole("status")).toContainText("Your password has been updated")
  expect(submitted).toEqual({ token, password: "new-local-password-123", passwordConfirm: "new-local-password-123" })
  expect(requestURL).not.toContain(token)
})

test("expired reset links offer a working request path", async ({ page }) => {
  await prepareAuthPage(page)
  await page.goto("/auth/reset-password")
  await expect(page.getByRole("status")).toContainText("This link is invalid or has expired")
  await expect(page.getByLabel("Email")).toBeVisible()

  await page.route("**/api/collections/users/confirm-password-reset", route => route.fulfill({
    status: 400,
    contentType: "application/json",
    body: JSON.stringify({ data: {}, message: "Invalid or expired token", status: 400 }),
  }))
  await page.goto("/auth/reset-password#expired-reset-token")
  const passwords = page.locator("input[autocomplete=new-password]")
  await passwords.nth(0).fill("new-local-password-123")
  await passwords.nth(1).fill("new-local-password-123")
  await page.getByRole("button", { name: "Update password" }).click()
  await expect(page.getByRole("alert")).toContainText("This link is invalid or has expired")
  await page.getByRole("link", { name: "Request a new password-reset link" }).click()
  await expect(page.getByRole("heading", { name: "Forgot password?" })).toBeVisible()
  await expect(page.getByLabel("Email")).toBeVisible()
  await expect(page.getByLabel("Email")).toBeFocused()
})
test("email-change confirmation sends its fragment token and current password in the body", async ({ page }) => {
  await prepareAuthPage(page)
  const token = "email-change-token-for-e2e"
  let submitted: Record<string, unknown> | undefined
  let requestURL = ""
  await page.route("**/api/collections/users/confirm-email-change", async route => {
    submitted = route.request().postDataJSON() as Record<string, unknown>
    requestURL = route.request().url()
    await route.fulfill({ status: 204 })
  })

  await page.goto("/auth/confirm-email-change#" + token)
  await expect(page.getByLabel("Current password")).toBeVisible()
  await expect.poll(() => page.evaluate(() => window.location.hash)).toBe("")
  expect(page.url()).not.toContain(token)
  await page.getByLabel("Current password").fill("current-local-password-123")
  await page.getByRole("button", { name: "Confirm email change" }).click()

  await expect(page).toHaveURL(/\/auth\?notice=email-changed/)
  await expect(page.getByRole("status")).toContainText("Your sign-in email has changed")
  expect(submitted).toEqual({ token, password: "current-local-password-123" })
  expect(requestURL).not.toContain(token)
})

test("profile can request an email change with an authenticated account", async ({ page, request }) => {
  const user = await registerAndAuthenticate(request)
  await prepareAuthPage(page, user.auth)
  const submitted: Array<Record<string, unknown>> = []
  await page.route("**/api/collections/users/request-email-change", async route => {
    submitted.push(route.request().postDataJSON() as Record<string, unknown>)
    await route.fulfill({ status: 204 })
  })

  await page.goto("/profile")
  const newEmail = "changed-" + randomUUID() + "@example.test"
  await page.getByLabel("New sign-in email").fill(newEmail)
  await page.getByRole("button", { name: "Send email-change confirmation" }).click()

  await expect(page.getByRole("status")).toContainText("A confirmation email was sent to the new address")
  expect(submitted).toEqual([{ newEmail }])
})

test("password MFA sends and resends the bound challenge, then accepts a pasted one-time code", async ({ page, request }) => {
  const user = await registerAndAuthenticate(request)
  await prepareAuthPage(page)
  const mfaId = "password-mfa-challenge"
  const otpRequests: Array<Record<string, unknown>> = []
  const authFlowPaths: string[] = []
  page.on("request", value => { const path = new URL(value.url()).pathname; if (path.includes("auth-with") || path.includes("otp")) authFlowPaths.push(path) })
  let loginBody: Record<string, unknown> | undefined
  await page.route("**/api/collections/users/auth-with-password", async route => {
    loginBody = route.request().postDataJSON() as Record<string, unknown>
    await route.fulfill({
      status: 401,
      contentType: "application/json",
      body: JSON.stringify({ data: { mfaId }, message: "Please authenticate with another method.", status: 401 }),
    })
  })
  await page.route("**/api/collections/users/request-otp", async route => {
    otpRequests.push(route.request().postDataJSON() as Record<string, unknown>)
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify({ otpId: "mfa-otp-" + otpRequests.length }),
    })
  })
  let otpLogin: Record<string, unknown> | undefined
  let otpLoginUrl = ""
  await page.route("**/api/collections/users/auth-with-otp*", async route => {
    otpLogin = route.request().postDataJSON() as Record<string, unknown>
    otpLoginUrl = route.request().url()
    await route.fulfill({
      status: 200,
      contentType: "application/json",
      body: JSON.stringify(user.auth),
    })
  })
  await page.route("**/api/subflow/v1/system/access*", route => route.fulfill({
    status: 200,
    contentType: "application/json",
    body: JSON.stringify({ data: { permissions: [] } }),
  }))

  await page.goto("/auth")
  await page.getByLabel("Email").fill(user.email)
  await page.locator("input[autocomplete=current-password]").fill(user.password)
  await page.getByRole("button", { name: "Log in" }).click()
  await expect(page.getByRole("heading", { name: "Second-step verification" })).toBeVisible()
  const code = page.getByLabel("Verification code")
  await expect(code).toHaveAttribute("autocomplete", "one-time-code")
  await code.fill("123456")
  await page.getByRole("button", { name: "Resend code" }).click()
  expect(otpRequests).toEqual([{ mfaId }, { mfaId }])
  await page.getByRole("button", { name: "Verify" }).click()

  await expect(page).toHaveURL(/\/home$/)
  expect(loginBody).toMatchObject({ identity: user.email, password: user.password })
  expect(otpLogin).toMatchObject({ otpId: "mfa-otp-2", password: "123456" })
  expect(new URL(otpLoginUrl).searchParams.get("mfaId")).toBe(mfaId)
})