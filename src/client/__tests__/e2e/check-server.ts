import http from 'node:http'

/**
 * Check if the gkill server (localhost:9999) is reachable.
 * Returns true if reachable, false otherwise.
 */
export function checkGkillServer(): Promise<boolean> {
  return new Promise((resolve) => {
    const req = http.request(
      { hostname: '127.0.0.1', port: 9999, path: '/', method: 'HEAD', timeout: 3000 },
      () => resolve(true),
    )
    req.on('error', () => resolve(false))
    req.on('timeout', () => { req.destroy(); resolve(false) })
    req.end()
  })
}
