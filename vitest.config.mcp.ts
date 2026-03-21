import { defineConfig } from 'vitest/config'

// Strip shebang lines so vitest can parse .mjs entry-point scripts.
function stripShebangPlugin() {
  return {
    name: 'strip-shebang',
    transform(code: string, id: string) {
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
    include: ['src/mcp/__tests__/**/*.test.mjs'],
    globals: true,
  },
})
