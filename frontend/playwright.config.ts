import { defineConfig, devices } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  fullyParallel: false,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 1 : 0,
  workers: 1,
  reporter: [['list'], ['html', { outputFolder: 'playwright-report', open: 'never' }]],
  outputDir: 'test-results',
  use: {
    baseURL: 'http://127.0.0.1:4174',
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    video: 'retain-on-failure',
    colorScheme: 'light',
  },
  projects: [
    { name: 'chromium', use: { ...devices['Desktop Chrome'], timezoneId: 'Asia/Taipei' } },
    { name: 'firefox', use: { ...devices['Desktop Firefox'], timezoneId: 'Asia/Taipei' } },
    { name: 'webkit', use: { ...devices['Desktop Safari'], timezoneId: 'Asia/Taipei' } },
    { name: 'chromium-touch', use: { ...devices['Pixel 5'], browserName: 'chromium', timezoneId: 'Asia/Taipei' } },
    { name: 'webkit-touch', use: { ...devices['iPhone 13'], browserName: 'webkit', timezoneId: 'Asia/Taipei' } },
  ],
  webServer: {
    command: 'node e2e/start.mjs',
    url: 'http://127.0.0.1:4174/api/health',
    reuseExistingServer: false,
    timeout: 120_000,
    stdout: 'pipe',
    stderr: 'pipe',
  },
})
