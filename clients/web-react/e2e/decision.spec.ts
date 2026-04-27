import { test, expect } from '@playwright/test'

test('decision modal opens and can be confirmed', async ({ page }) => {
  test.setTimeout(60_000)

  // Use a kingdom with chapel so a decision is likely to appear
  await page.goto('/new')
  // manually select chapel + 9 others
  await page.getByTestId('card-tile-chapel').click()
  await page.getByTestId('card-tile-smithy').click()
  await page.getByTestId('card-tile-village').click()
  await page.getByTestId('card-tile-market').click()
  await page.getByTestId('card-tile-laboratory').click()
  await page.getByTestId('card-tile-festival').click()
  await page.getByTestId('card-tile-cellar').click()
  await page.getByTestId('card-tile-mine').click()
  await page.getByTestId('card-tile-witch').click()
  await page.getByTestId('card-tile-moat').click()
  await page.getByTestId('start-game-btn').click()
  await expect(page).toHaveURL(/\/game\/.+/)

  // Play until a decision modal appears or 30 end-phase clicks
  let clicks = 0
  while (clicks < 30) {
    const modal = page.getByTestId('decision-modal')
    const endBtn = page.getByTestId('end-phase-btn')

    const which = await Promise.race([
      modal.waitFor({ state: 'visible', timeout: 3000 }).then(() => 'modal' as const),
      endBtn.waitFor({ state: 'visible', timeout: 3000 }).then(() => 'btn' as const),
    ]).catch(() => 'timeout' as const)

    if (which === 'modal') {
      // Modal is open — confirm it
      const confirmBtn = page.getByTestId('decision-confirm-btn')
      const yesBtn = page.getByTestId('decision-yes-btn')
      if (await confirmBtn.isVisible()) {
        await confirmBtn.click()
      } else if (await yesBtn.isVisible()) {
        await yesBtn.click()
      }
      await expect(modal).not.toBeVisible({ timeout: 5000 })
      break
    }
    if (which === 'btn') {
      await endBtn.click()
      clicks++
    } else {
      break
    }
  }
})
