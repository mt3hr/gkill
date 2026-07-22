import http from 'node:http'

// GKILL_E2E_BASE_URL でテスト対象サーバを上書きできる (既定: http://localhost:9999)
const baseUrl = new URL(process.env.GKILL_E2E_BASE_URL ?? 'http://localhost:9999')
const gkillPort = Number(baseUrl.port || 9999)

// GKILL_E2E_VITE_URL でVite dev serverを上書きできる (既定: http://localhost:5173)
const viteUrl = new URL(process.env.GKILL_E2E_VITE_URL ?? 'http://localhost:5173')
const vitePort = Number(viteUrl.port || 5173)

/**
 * Check if the gkill server is reachable.
 * Returns true if reachable, false otherwise.
 */
export function checkGkillServer(): Promise<boolean> {
  return new Promise((resolve) => {
    const req = http.request(
      { hostname: baseUrl.hostname, port: gkillPort, path: '/', method: 'GET', timeout: 10000 },
      () => resolve(true),
    )
    req.on('error', () => resolve(false))
    req.on('timeout', () => { req.destroy(); resolve(false) })
    req.end()
  })
}

// 一度到達できたらそれを覚えておく。ワーカー再起動のたびにプローブし直すと、
// 負荷が高いタイミングでタイムアウトしてテストが黙ってskipされるため
let viteApiReachable = false

function probeGkillApiViaVite(timeoutMs: number): Promise<boolean> {
  return new Promise((resolve) => {
    const req = http.request(
      { hostname: viteUrl.hostname, port: vitePort, path: '/api/login', method: 'POST', timeout: timeoutMs,
        headers: { 'Content-Type': 'application/json' } },
      (res) => {
        // If Vite proxies to gkill, we get a JSON response (200 or error with JSON body).
        // If Vite doesn't proxy, we get 404 or HTML.
        const ct = res.headers['content-type'] || ''
        res.resume()
        resolve(ct.includes('application/json') || res.statusCode === 200)
      },
    )
    req.on('error', () => resolve(false))
    req.on('timeout', () => { req.destroy(); resolve(false) })
    req.write(JSON.stringify({ user_id: '', password_sha256: '' }))
    req.end()
  })
}

/**
 * Check if the gkill API is reachable via Vite dev server.
 * Returns true if /api/ is proxied to gkill server, false otherwise.
 * 負荷でのタイムアウトを誤検知しないよう、30秒プローブを3回まで試す。
 */
export async function checkGkillApiViaVite(): Promise<boolean> {
  if (viteApiReachable) {
    return true
  }
  for (let attempt = 0; attempt < 3; attempt++) {
    if (await probeGkillApiViaVite(30000)) {
      viteApiReachable = true
      return true
    }
  }
  return false
}
