import { test, expect } from '@playwright/test'

test('lobby shows Create Game button', async ({ page }) => {
  await page.goto('/')
  await expect(page.getByTestId('create-game-btn')).toBeVisible()
})

test('Create Game button navigates to /new', async ({ page }) => {
  await page.goto('/')
  await page.getByTestId('create-game-btn').click()
  await expect(page).toHaveURL('/new')
  await expect(page.getByTestId('kingdom-grid')).toBeVisible()
})
