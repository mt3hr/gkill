/**
 * E2E test runner: starts gkill_server with test home, runs Playwright, then cleans up.
 */
import { execSync, spawn } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import http from 'node:http'

const home = process.env.HOME || process.env.USERPROFILE || ''
const testHome = path.join(home, 'gkill_test')

// 1. Clean test home directory
try {
  if (fs.existsSync(testHome)) {
    fs.rmSync(testHome, { recursive: true, force: true })
  }
} catch {
  // ignore — may have locked files from previous run
}
fs.mkdirSync(testHome, { recursive: true })

// 2. Start gkill_server with test home
console.log('[E2E] Starting gkill_server with --gkill_home_dir', testHome)
const server = spawn('gkill_server', [
  '--gkill_home_dir', testHome,
  '--disable_tls',
  '--log', 'none',
], {
  stdio: ['ignore', 'pipe', 'pipe'],
  detached: false,
})

server.stdout.on('data', (d) => process.stdout.write(`[gkill_server] ${d}`))
server.stderr.on('data', (d) => process.stderr.write(`[gkill_server] ${d}`))

// 3. Wait for server to be ready
async function waitForServer(url, timeoutMs = 30000) {
  const start = Date.now()
  while (Date.now() - start < timeoutMs) {
    try {
      await new Promise((resolve, reject) => {
        const req = http.request(url, { method: 'GET', timeout: 3000 }, () => resolve())
        req.on('error', reject)
        req.on('timeout', () => { req.destroy(); reject(new Error('timeout')) })
        req.end()
      })
      return true
    } catch {
      await new Promise(r => setTimeout(r, 500))
    }
  }
  return false
}

const ready = await waitForServer('http://127.0.0.1:9999/')
if (!ready) {
  console.error('[E2E] gkill_server failed to start within 30 seconds')
  server.kill()
  process.exit(1)
}
console.log('[E2E] gkill_server is ready')

// 4. Run Playwright tests
let exitCode = 0
try {
  execSync('npx playwright test', { stdio: 'inherit', cwd: process.cwd() })
} catch (e) {
  exitCode = e.status || 1
}

// 5. Stop gkill_server
console.log('[E2E] Stopping gkill_server')
server.kill()

// Give it a moment to release files
await new Promise(r => setTimeout(r, 1000))

process.exit(exitCode)
