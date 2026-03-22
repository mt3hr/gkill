/**
 * Playwright global teardown.
 * Playwright automatically stops webServer processes, so no manual cleanup needed.
 */
export default async function globalTeardown() {
  // Playwright handles stopping gkill_server and Vite dev server automatically.
}
