import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [
    vue({
      script: {
        propsDestructure: true,
      },
    }),
  ],
  test: {
    environment: 'jsdom',
    include: ['src/client/__tests__/unit/**/*.test.ts'],
    globals: true,
    // Dnoteの集計はローカル暦の日・週・月でバケットを切るため、
    // 実行環境のタイムゾーンで結果が変わる。
    // trend-aggregator のテストデータは +09:00 前提で書かれており、
    // UTCで走らせるとバケットが1つ増えて落ちる（CIで実際に落ちた）。
    // 誰がどこで回しても同じ結果になるよう固定する。
    env: {
      TZ: 'Asia/Tokyo',
    },
  },
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src/client', import.meta.url)),
    },
  },
})
