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
    // **/api/login を叩いてはいけない。** handle_login.go は資格情報の検証より前に
    // ログインのレート制限（IP毎 15分に10回）を1回消費する。
    // このプローブのキャッシュはワーカープロセス単位で、Playwright はテスト失敗のたびに
    // ワーカーを作り直すので、失敗が続くとプローブが繰り返されて枠を食い潰し、
    // 本物のログイン（setup / login.spec.ts / ログアウトのテスト）まで巻き添えで落ちる。
    // get_application_config は wrapAuth なので未認証でも
    // Content-Type: application/json のエラー応答が返る（レート制限は無い）。
    const req = http.request(
      { hostname: viteUrl.hostname, port: vitePort, path: '/api/get_application_config', method: 'POST', timeout: timeoutMs,
        headers: { 'Content-Type': 'application/json' } },
      (res) => {
        // Vite が gkill へプロキシしていれば JSON が返る。
        // プロキシしていなければ 404 か HTML が返る
        const ct = res.headers['content-type'] || ''
        res.resume()
        resolve(ct.includes('application/json'))
      },
    )
    req.on('error', () => resolve(false))
    req.on('timeout', () => { req.destroy(); resolve(false) })
    req.write(JSON.stringify({ session_id: '', locale_name: 'ja' }))
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
