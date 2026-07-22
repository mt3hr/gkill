import { defineConfig } from '@playwright/test'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const STORAGE_STATE = path.join(__dirname, 'src/client/__tests__/e2e/.auth/user.json')

export default defineConfig({
  testDir: 'src/client/__tests__/e2e',
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  // リトライで通ったテストはpassedではなくflakyとして別集計されるので、
  // フレークを隠さずに件数を可視化したまま実行できる
  retries: process.env.CI ? 2 : 1,
  // 全ワーカーが1台のSQLiteバックエンドに集中するため、既定 (CPU数の半分) だと
  // 負荷でタイムアウトする。4に絞る
  workers: process.env.CI ? 1 : 4,
  // open: 'never' にしないと失敗時にレポートサーバ(:9323)が立ち上がってrun-e2e.mjsがブロックされる
  reporter: [['html', { open: 'never' }]],
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
