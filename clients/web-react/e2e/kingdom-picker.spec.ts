import { test, expect } from '@playwright/test'

test('Randomize selects exactly 10 cards', async ({ page }) => {
  await page.goto('/new')
  await page.getByTestId('randomize-btn').click()
  const startBtn = page.getByTestId('start-game-btn')
  await expect(startBtn).toBeEnabled()
  // count visually-selected tiles (have brightness-75 class)
  const selected = page.locator('[data-testid^="card-tile-"] img.brightness-75')
  await expect(selected).toHaveCount(10)
})

test('manual picker enforces exactly 10', async ({ page }) => {
  await page.goto('/new')
  const startBtn = page.getByTestId('start-game-btn')
  await expect(startBtn).toBeDisabled()

  // select 10 kingdom cards
  const tiles = page.locator('[data-testid^="card-tile-"]')
  for (let i = 0; i < 10; i++) {
    await tiles.nth(i).click()
  }
  await expect(startBtn).toBeEnabled()

  // deselect one — button disables again
  await tiles.nth(0).click()
  await expect(startBtn).toBeDisabled()
})

test('Start Game creates game and navigates to /game/:id', async ({ page }) => {
  await page.goto('/new')
  await page.getByTestId('randomize-btn').click()
  await page.getByTestId('start-game-btn').click()
  await expect(page).toHaveURL(/\/game\/.+/)
  await expect(page.getByTestId('supply-grid')).toBeVisible()
})
