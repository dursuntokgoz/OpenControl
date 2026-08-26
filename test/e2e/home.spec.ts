import { expect, test } from '@playwright/test'

test.describe('panel shell', () => {
  test('user dashboard shows healthy server', async ({ page }) => {
    await page.goto('/')
    await expect(page.getByTestId('app-title')).toHaveText('ServerPanel')
    await expect(page.getByTestId('health-status')).toHaveText('ok', { timeout: 15_000 })
    await expect(page.getByTestId('health-version')).toBeVisible()
  })

  test('admin route serves the SPA and reports health', async ({ page }) => {
    await page.goto('/admin')
    // Client-side route must render through index.html fallback.
    await expect(page.getByTestId('app-title')).toHaveText('ServerPanel')
    await expect(page.getByTestId('health-status')).toHaveText('ok', { timeout: 15_000 })
  })

  test('language switcher persists Turkish selection', async ({ page }) => {
    await page.goto('/admin')
    await page.getByRole('button', { name: 'tr' }).click()
    await expect(page.getByRole('heading', { name: 'Sunucu yönetim konsolu' })).toBeVisible()
    await page.reload()
    await expect(
      page.getByRole('heading', { name: 'Sunucu yönetim konsolu' }),
    ).toBeVisible()
  })
})
