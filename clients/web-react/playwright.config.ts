import { defineConfig } from '@playwright/test'

export default defineConfig({
  testDir: './e2e',
  timeout: 30_000,
  use: {
    baseURL: 'http://localhost:5173',
    headless: true,
  },
  webServer: [
    {
      command: 'make -C ../.. server',
      url: 'http://localhost:8080/healthz',
      reuseExistingServer: true,
      timeout: 10_000,
    },
    {
      command: 'pnpm run dev',
      url: 'http://localhost:5173',
      reuseExistingServer: true,
      timeout: 15_000,
    },
  ],
  projects: [{ name: 'chromium', use: { browserName: 'chromium' } }],
})
