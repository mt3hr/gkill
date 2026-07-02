import { defineConfig } from '@playwright/test'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const STORAGE_STATE = path.join(__dirname, 'src/client/__tests__/e2e/.auth/user.json')

export default defineConfig({
  testDir: 'src/client/__tests__/e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: process.env.CI ? 2 : 0,
  workers: process.env.CI ? 1 : undefined,
  reporter: 'html',
  timeout: 60000,
  globalSetup: './src/client/__tests__/e2e/global-setup.ts',
  globalTeardown: './src/client/__tests__/e2e/global-teardown.ts',
  use: {
    // GKILL_E2E_BASE_URL でテスト対象サーバを上書きできる (別ポートでの実行用)
    baseURL: process.env.GKILL_E2E_BASE_URL ?? 'http://localhost:9999',
    trace: 'on-first-retry',
    navigationTimeout: 60000,
    actionTimeout: 10000,
  },
  projects: [
    { name: 'setup', testMatch: /auth\.setup\.ts/ },
    {
      name: 'default',
      dependencies: ['setup'],
      use: { storageState: STORAGE_STATE },
      testIgnore: /auth\.setup\.ts/,
    },
  ],
})
