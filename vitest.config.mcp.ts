import { defineConfig } from 'vitest/config'

export default defineConfig({
  test: {
    environment: 'node',
    include: ['src/mcp/__tests__/**/*.test.mjs'],
    globals: true,
  },
})
