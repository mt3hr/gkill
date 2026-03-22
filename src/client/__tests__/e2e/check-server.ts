import http from 'node:http'

/**
 * Check if the gkill server (localhost:9999) is reachable.
 * Returns true if reachable, false otherwise.
 */
export function checkGkillServer(): Promise<boolean> {
  return new Promise((resolve) => {
    const req = http.request(
      { hostname: '127.0.0.1', port: 9999, path: '/', method: 'GET', timeout: 10000 },
      () => resolve(true),
    )
    req.on('error', () => resolve(false))
    req.on('timeout', () => { req.destroy(); resolve(false) })
    req.end()
  })
}

/**
 * Check if the gkill API is reachable via Vite dev server (localhost:5173).
 * Returns true if /api/ is proxied to gkill server, false otherwise.
 */
export function checkGkillApiViaVite(): Promise<boolean> {
  return new Promise((resolve) => {
    const req = http.request(
      { hostname: '127.0.0.1', port: 5173, path: '/api/login', method: 'POST', timeout: 5000,
        headers: { 'Content-Type': 'application/json' } },
      (res) => {
        // If Vite proxies to gkill, we get a JSON response (200 or error with JSON body).
        // If Vite doesn't proxy, we get 404 or HTML.
        const ct = res.headers['content-type'] || ''
        resolve(ct.includes('application/json') || res.statusCode === 200)
      },
    )
    req.on('error', () => resolve(false))
    req.on('timeout', () => { req.destroy(); resolve(false) })
    req.write(JSON.stringify({ user_id: '', password_sha256: '' }))
    req.end()
  })
}
