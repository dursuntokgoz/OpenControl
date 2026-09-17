import { expect, test } from '@playwright/test'

const ADMIN_USER = 'admin'
const ADMIN_PASS = 'admin1234'

async function login(page: import('@playwright/test').Page) {
  await page.goto('/login')
  await page.getByLabel('Username').fill(ADMIN_USER)
  await page.getByLabel('Password').fill(ADMIN_PASS)
  await page.getByRole('button', { name: 'Sign in' }).click()
  // Wait for redirect away from login page
  await expect(page).not.toHaveURL(/\/login/)
}

test.describe('authentication', () => {
  test('login page renders and accepts credentials', async ({ page }) => {
    await page.goto('/login')
    await expect(page.getByRole('heading', { name: 'ServerPanel' })).toBeVisible()
    await page.getByLabel('Username').fill(ADMIN_USER)
    await page.getByLabel('Password').fill(ADMIN_PASS)
    await page.getByRole('button', { name: 'Sign in' }).click()
    await expect(page).not.toHaveURL(/\/login/, { timeout: 10_000 })
  })

  test('login rejects wrong password', async ({ page }) => {
    await page.goto('/login')
    await page.getByLabel('Username').fill(ADMIN_USER)
    await page.getByLabel('Password').fill('wrongpassword')
    await page.getByRole('button', { name: 'Sign in' }).click()
    await expect(page.getByRole('alert')).toBeVisible({ timeout: 5_000 })
  })
})

test.describe('panel shell (authenticated)', () => {
  test('user dashboard shows healthy server', async ({ page }) => {
    await login(page)
    await page.goto('/')
    await expect(page.getByTestId('app-title')).toHaveText('ServerPanel')
    await expect(page.getByTestId('health-status')).toHaveText('ok', { timeout: 15_000 })
    await expect(page.getByTestId('health-version')).toBeVisible()
  })

  test('admin route serves the SPA and reports health', async ({ page }) => {
    await login(page)
    await page.goto('/admin')
    await expect(page.getByTestId('app-title')).toHaveText('ServerPanel')
    await expect(page.getByTestId('health-status')).toHaveText('ok', { timeout: 15_000 })
  })

  test('language switcher persists Turkish selection', async ({ page }) => {
    await login(page)
    await page.goto('/admin')
    await page.getByRole('button', { name: 'tr' }).click()
    await expect(page.getByRole('heading', { name: 'Sunucu yönetim konsolu' })).toBeVisible()
    await page.reload()
    await expect(
      page.getByRole('heading', { name: 'Sunucu yönetim konsolu' }),
    ).toBeVisible()
  })
})

test.describe('unauthenticated redirect', () => {
  test('root redirects to login', async ({ page }) => {
    await page.goto('/')
    await expect(page).toHaveURL(/\/login/)
  })

  test('admin redirects to login', async ({ page }) => {
    await page.goto('/admin')
    await expect(page).toHaveURL(/\/login/)
  })
})
