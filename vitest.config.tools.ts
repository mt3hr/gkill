import { defineConfig } from 'vitest/config'

// src/tools/ の .mjs は先頭に shebang を持つものがあるので、import できるよう剥がす
// （vitest.config.mcp.ts と同じ手当て）。
function stripShebangPlugin() {
  return {
    name: 'strip-shebang',
    transform(code: string, _id: string) {
      if (code.startsWith('#!')) {
        return { code: code.replace(/^#![^\n]*\n/, '\n'), map: null }
      }
    },
  }
}

export default defineConfig({
  plugins: [stripShebangPlugin()],
  test: {
    environment: 'node',
    include: ['src/tools/__tests__/**/*.test.mjs'],
    globals: true,
    // 一時 git リポジトリを作って git を何度も呼ぶので、既定の 5 秒では足りないことがある
    testTimeout: 30000,
  },
})
