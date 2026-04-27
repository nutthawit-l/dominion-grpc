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
  test.setTimeout(120_000)
  await createAndJoin(page)

  // Keep clicking End Phase whenever it appears, until game ends
  const maxClicks = 200
  let clicks = 0
  while (clicks < maxClicks) {
    const endBtn = page.getByTestId('end-phase-btn')
    const gameOver = page.getByText(/game over/i)

    const which = await Promise.race([
      endBtn.waitFor({ state: 'visible', timeout: 8000 }).then(() => 'btn' as const),
      gameOver.waitFor({ state: 'visible', timeout: 8000 }).then(() => 'over' as const),
    ]).catch(() => 'timeout' as const)

    if (which === 'over') break
    if (which === 'timeout') break
    await endBtn.click()
    clicks++
  }

  await expect(page.getByText(/game over/i)).toBeVisible({ timeout: 5000 })
})
