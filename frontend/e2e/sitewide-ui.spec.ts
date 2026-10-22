import { expect, test, type Page } from '@playwright/test'
import { mkdir } from 'node:fs/promises'
import { dirname } from 'node:path'

async function signInAsTestOwner(page: Page) {
  await page.addInitScript(() => localStorage.setItem('subflow-locale', 'en'))
  await page.goto('/auth')
  await page.getByLabel('Email').fill('owner@example.test')
  await page.locator('input[autocomplete="current-password"]').fill('test-password-1234')
  await page.locator('form').getByRole('button', { name: 'Log in' }).click()
  await expect(page).toHaveURL(/personal/)
  await page.getByRole('button', { name: 'Not now' }).click({ timeout: 1500 }).catch(() => {})
}

async function expectNoHorizontalOverflow(page: Page) {
  const sizes = await page.evaluate(() => ({ viewport: document.documentElement.clientWidth, content: document.documentElement.scrollWidth }))
  expect(sizes.content, JSON.stringify(sizes)).toBeLessThanOrEqual(sizes.viewport + 1)
}

test('all application routes render on desktop and narrow screens', async ({ page }, testInfo) => {
  test.setTimeout(120_000)
  await signInAsTestOwner(page)

  await page.goto('/personal')
  const addRecord = page.locator('.ledger-add-desktop')
  await addRecord.click()
  let expenseDrawer = page.getByRole('dialog', { name: 'Add record' })
  await expenseDrawer.getByRole('button', { name: 'Save record' }).click()
  await expect(expenseDrawer.getByLabel('Item')).toBeFocused()
  await expenseDrawer.getByRole('button', { name: 'Cancel' }).click()
  await expect(expenseDrawer).toBeHidden()

  await addRecord.click()
  expenseDrawer = page.getByRole('dialog', { name: 'Add record' })
  await expenseDrawer.getByLabel('Item').fill('E2E expense original')
  await expenseDrawer.getByLabel('Amount').fill('123')
  await expenseDrawer.getByRole('button', { name: 'Save record' }).click()
  await expect(expenseDrawer).toBeHidden()
  let expenseRow = page.locator('.ledger-item').filter({ hasText: 'E2E expense original' })
  await expect(expenseRow).toBeVisible()
  await expenseRow.getByRole('button', { name: 'Edit' }).click()
  expenseDrawer = page.getByRole('dialog', { name: 'Edit expense' })
  await expenseDrawer.getByLabel('Item').fill('E2E expense updated')
  await expenseDrawer.getByRole('button', { name: 'Save record' }).click()
  await expect(page.locator('.ledger-item').filter({ hasText: 'E2E expense updated' })).toBeVisible()
  expenseRow = page.locator('.ledger-item').filter({ hasText: 'E2E expense updated' })
  await expenseRow.getByRole('button', { name: 'Remove' }).click()
  let confirmation = page.getByRole('alertdialog')
  await confirmation.getByRole('button', { name: 'Cancel' }).click()
  await expect(confirmation).toBeHidden()
  await expect(expenseRow).toBeVisible()
  await expenseRow.getByRole('button', { name: 'Remove' }).click()
  confirmation = page.getByRole('alertdialog')
  await confirmation.getByRole('button', { name: 'Delete' }).click()
  await expect(expenseRow).toBeHidden()

  await page.goto('/groups')
  await page.getByRole('button', { name: 'Create group' }).click()
  const createDialog = page.getByRole('dialog', { name: 'Create group' })
  await createDialog.getByRole('button', { name: 'Cancel' }).click()
  await expect(createDialog).toBeHidden()
  await page.getByRole('button', { name: 'Create group' }).click()
  const reopenedDialog = page.getByRole('dialog', { name: 'Create group' })
  await reopenedDialog.getByRole('button', { name: 'Create group' }).click()
  await expect(reopenedDialog.getByLabel('Group name')).toBeFocused()
  await expect(reopenedDialog.getByLabel('Group name')).toHaveJSProperty('validity.valid', false)
  await reopenedDialog.getByLabel('Group name').fill('Responsive test group')
  await reopenedDialog.getByRole('button', { name: 'Create group' }).click()
  await expect(page).toHaveURL(/\/groups\/[^/]+\/overview/)
  const groupPath = new URL(page.url()).pathname.split('/').slice(0, 3).join('/')

  await page.goto(`${groupPath}/settings`)
  const groupName = page.getByLabel('Group name')
  await expect(groupName).toHaveValue('Responsive test group')
  await groupName.fill('Responsive test group updated')
  await page.getByRole('button', { name: 'Save changes' }).click()
  await expect(groupName).toHaveValue('Responsive test group updated')

  const routes = [
    '/overview', '/groups', '/personal', '/personal/incomes', '/personal/expenses', '/personal/subscriptions',
    '/personal/share', '/profile', '/notifications', '/about', '/admin', '/admin/settings', '/admin/users',
    '/admin/roles', '/admin/audit', '/admin/notification-emails',
    `${groupPath}/overview`, `${groupPath}/ledger`, `${groupPath}/expenses`, `${groupPath}/incomes`,
    `${groupPath}/subscriptions`, `${groupPath}/members`, `${groupPath}/roles`, `${groupPath}/audit`,
    `${groupPath}/settings`, `${groupPath}/share`, '/invite/invalid-test-token', '/share/invalid-test-token',
  ]
  const widths = [1440, 1024, 768, 390, 375, 320]
  const screenshotsRoot = testInfo.outputPath('sitewide-routes')

  for (const [routeIndex, route] of routes.entries()) {
    await page.setViewportSize({ width: 1440, height: 1000 })
    await page.goto(route)
    await expect(page.locator('body')).not.toContainText('Unexpected error')
    await expectNoHorizontalOverflow(page)
    if (routeIndex === 2 || route.includes('/ledger')) {
      const themeButton = page.locator('.theme-switcher').first()
      if (await themeButton.isVisible()) {
        const size = await themeButton.boundingBox()
        expect(size?.width).toBe(44)
        expect(size?.height).toBe(44)
      }
    }
    const desktopPath = `${screenshotsRoot}/${String(routeIndex).padStart(2, '0')}-desktop.png`
    await mkdir(dirname(desktopPath), { recursive: true })
    await page.screenshot({ path: desktopPath, fullPage: true, animations: 'disabled' })

    await page.setViewportSize({ width: 375, height: 812 })
    await expectNoHorizontalOverflow(page)
    if (route === '/personal' || route.endsWith('/ledger')) {
      const addButton = page.locator('.ledger-fab').first()
      await expect(addButton).toBeVisible()
      const background = await addButton.evaluate(element => getComputedStyle(element).backgroundColor)
      expect(background).not.toBe('rgba(0, 0, 0, 0)')
    }
    const mobilePath = `${screenshotsRoot}/${String(routeIndex).padStart(2, '0')}-mobile.png`
    await page.screenshot({ path: mobilePath, fullPage: true, animations: 'disabled' })
  }

  for (const width of widths) {
    await page.setViewportSize({ width, height: 900 })
    await page.goto(`${groupPath}/audit`)
    await expectNoHorizontalOverflow(page)
    if (width <= 900) {
      const menu = page.locator('.group-mobile-nav')
      await expect(menu).toBeVisible()
      await menu.locator('summary').click()
      await expect(menu.getByRole('link', { name: 'Audit logs' })).toBeVisible()
    }
  }

  await page.goto('/personal?date=2026-01-01&keep=1')
  await expect(page.locator('.ledger-date-nav')).toBeVisible()
  await page.getByRole('button', { name: 'Previous day' }).click()
  await expect(page).toHaveURL(/date=2025-12-31/)
  await expect(page).toHaveURL(/keep=1/)
  const datePicker = page.getByLabel('Choose date')
  await datePicker.fill('2026-02-01')
  await expect(datePicker).toHaveValue('2026-02-01')
  await datePicker.dispatchEvent('change')
  await expect(page).toHaveURL(/date=2026-02-01/)
  await page.getByRole('button', { name: 'Switch to dark mode' }).click()
  await expect(page.locator('html')).toHaveAttribute('data-theme', 'dark')

  await page.goto(`${groupPath}/audit`)
  await page.getByRole('textbox', { name: 'Search audit logs' }).fill('no matching audit item')
  await page.getByRole('button', { name: 'Apply filters' }).click()
  await expect(page).toHaveURL(/q=no\+matching\+audit\+item/)
  await page.getByRole('button', { name: 'Clear filters' }).click()
  await expect(page).not.toHaveURL(/q=/)
})
