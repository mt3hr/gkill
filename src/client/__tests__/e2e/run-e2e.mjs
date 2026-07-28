/**
 * E2E test runner: starts gkill_server and Vite dev server on free ports,
 * runs Playwright, then cleans up.
 *
 * ポートは毎回空きポートを採番する。開発機では本番のgkill_serverが常駐して
 * :9999 を掴んでいることがあり、固定ポートだと起動できない・本番に向けて
 * テストが走ってしまうため。
 */
import { execFileSync, execSync, spawn } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import http from 'node:http'
import { getFreePorts } from './free-port.mjs'

const home = process.env.HOME || process.env.USERPROFILE || ''
const testHome = path.join(home, 'gkill_test')

// 0. Kill leftover gkill_server from previous E2E runs.
//    テストhome配下を指しているプロセスだけを対象にする (常駐している本番サーバは落とさない)
if (process.platform === 'win32') {
  // PowerShell側はシングルクォートだけで書く (cmd.exe経由の二重引用符エスケープを避けるため)
  const psCommand =
    "Get-CimInstance Win32_Process | " +
    "Where-Object { $_.Name -eq 'gkill_server.exe' -and $_.CommandLine -like '*gkill_test*' } | " +
    'ForEach-Object { Stop-Process -Id $_.ProcessId -Force }'
  try {
    execSync(`powershell -NoProfile -Command "${psCommand}"`, { stdio: 'ignore' })
  } catch { /* no process */ }
} else {
  try { execSync('pkill -f "gkill_server.*gkill_test"', { stdio: 'ignore' }) } catch { /* no process */ }
}
// Wait for file locks to release after killing
await new Promise(r => setTimeout(r, 2000))

// 1. Clean test home directory
try {
  if (fs.existsSync(testHome)) {
    fs.rmSync(testHome, { recursive: true, force: true })
  }
} catch {
  // ignore — may have locked files from previous run
}
fs.mkdirSync(testHome, { recursive: true })

// 2. Allocate free ports and start gkill_server with test home
const [gkillPort, vitePort] = await getFreePorts(2)
const gkillBaseUrl = `http://127.0.0.1:${gkillPort}`
const viteBaseUrl = `http://127.0.0.1:${vitePort}`

console.log(`[E2E] Starting gkill_server on ${gkillBaseUrl} with --gkill_home_dir ${testHome}`)
const server = spawn('gkill_server', [
  '--gkill_home_dir', testHome,
  '--address', `127.0.0.1:${gkillPort}`,
  '--disable_tls',
  '--log', 'none',
], {
  stdio: ['ignore', 'pipe', 'pipe'],
  detached: false,
})

server.stdout.on('data', (d) => process.stdout.write(`[gkill_server] ${d}`))
server.stderr.on('data', (d) => process.stderr.write(`[gkill_server] ${d}`))

let vite = null

function stopServers() {
  if (vite) {
    console.log('[E2E] Stopping Vite dev server')
    vite.kill()
    vite = null
  }
  if (server.exitCode === null) {
    console.log('[E2E] Stopping gkill_server')
    server.kill()
  }
}

// 3. Wait for a server to be ready
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

let exitCode = 0
try {
  if (!await waitForServer(`${gkillBaseUrl}/`)) {
    throw new Error('gkill_server failed to start within 30 seconds')
  }
  console.log('[E2E] gkill_server is ready')

  // 4. Start Vite dev server proxying /api to the test gkill_server.
  //    これを立てないとCRUD系のspecがskipされる。proxy先を明示することで
  //    本番の:9999に書き込んでしまう事故も防ぐ。
  //    シェル経由 (npx) だとWindowsでkillが子プロセスに届かないのでnodeで直接起動する
  console.log(`[E2E] Starting Vite dev server on ${viteBaseUrl}`)
  const viteBin = path.join(process.cwd(), 'node_modules', 'vite', 'bin', 'vite.js')
  vite = spawn(process.execPath, [viteBin, '--host', '127.0.0.1', '--port', String(vitePort), '--strictPort'], {
    stdio: ['ignore', 'pipe', 'pipe'],
    detached: false,
    env: { ...process.env, GKILL_API_PROXY_TARGET: gkillBaseUrl },
  })
  vite.stdout.on('data', (d) => process.stdout.write(`[vite] ${d}`))
  vite.stderr.on('data', (d) => process.stderr.write(`[vite] ${d}`))

  if (!await waitForServer(`${viteBaseUrl}/`, 60000)) {
    throw new Error('Vite dev server failed to start within 60 seconds')
  }
  console.log('[E2E] Vite dev server is ready')

  // 5. Run Playwright tests (追加引数はそのままplaywrightに渡す: npm run test_client_e2e -- foo.spec.ts --workers=1)
  const playwrightArgs = process.argv.slice(2)
  try {
    // シェルを経由させないため、Vite起動と同じくnodeでCLIを直接叩く。
    // npx経由だと追加引数がそのままシェルに解釈される
    const playwrightBin = path.join(process.cwd(), 'node_modules', '@playwright', 'test', 'cli.js')
    execFileSync(process.execPath, [playwrightBin, 'test', ...playwrightArgs], {
      stdio: 'inherit',
      cwd: process.cwd(),
      env: {
        ...process.env,
        GKILL_E2E_BASE_URL: gkillBaseUrl,
        GKILL_E2E_VITE_URL: viteBaseUrl,
      },
    })
  } catch (e) {
    exitCode = e.status || 1
  }
} catch (e) {
  console.error(`[E2E] ${e.message}`)
  exitCode = 1
} finally {
  // 6. Stop servers
  stopServers()
}

// Give it a moment to release files
await new Promise(r => setTimeout(r, 1000))

process.exit(exitCode)
