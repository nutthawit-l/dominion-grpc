import { test, expect } from '@playwright/test'

async function createAndJoin(page: import('@playwright/test').Page) {
  await page.goto('/new')
  await page.getByTestId('randomize-btn').click()
  await page.getByTestId('start-game-btn').click()
  await expect(page).toHaveURL(/\/game\/.+/)
}

test('game table shows supply grid after creation', async ({ page }) => {
  await createAndJoin(page)
  await expect(page.getByTestId('supply-grid')).toBeVisible()
  await expect(page.getByTestId('game-log')).toBeVisible()
  await expect(page.getByTestId('card-area')).toBeVisible()
})

test('End Phase button is visible on my turn', async ({ page }) => {
  await createAndJoin(page)
  // player 0 goes first — End Phase should be available
  await expect(page.getByTestId('end-phase-btn')).toBeVisible({ timeout: 5000 })
})

test('human can end phases until game over', async ({ page }) => {
  test.setTimeout(300_000)
  await createAndJoin(page)

  // Keep clicking End Phase whenever it appears, until game ends.
  // Use page.click with a short timeout so detached-element errors don't stall us.
  const maxClicks = 400
  let clicks = 0
  while (clicks < maxClicks) {
    const gameOver = page.getByText(/game over/i)
    if (await gameOver.isVisible()) break

    try {
      // Short timeout: if the button isn't clickable quickly, loop and try again.
      await page.click('[data-testid="end-phase-btn"]', { timeout: 5000 })
      clicks++
    } catch {
      // Button may have disappeared (bot turn) or detached mid-click — check for game over
      if (await gameOver.isVisible()) break
      // Small pause before retrying to let React settle
      await page.waitForTimeout(200)
    }
  }

  await expect(page.getByText(/game over/i)).toBeVisible({ timeout: 15_000 })
})
