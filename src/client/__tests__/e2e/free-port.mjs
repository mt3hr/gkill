/**
 * 空きポート採番。固定ポートだと開発機で常駐しているgkill_serverと衝突するため、
 * E2Eで起動するサーバのポートは毎回OSに選ばせる。
 */
import net from 'node:net'

/**
 * OSに空きポートを1つ選ばせて返す。
 * @param {string} host バインドするホスト
 * @returns {Promise<number>}
 */
export function getFreePort(host = '127.0.0.1') {
  return new Promise((resolve, reject) => {
    const server = net.createServer()
    server.unref()
    server.on('error', reject)
    server.listen(0, host, () => {
      const { port } = server.address()
      server.close(() => resolve(port))
    })
  })
}

/**
 * 空きポートを重複なくn個返す。
 * @param {number} count
 * @param {string} host
 * @returns {Promise<number[]>}
 */
export async function getFreePorts(count, host = '127.0.0.1') {
  const ports = []
  while (ports.length < count) {
    const port = await getFreePort(host)
    if (!ports.includes(port)) {
      ports.push(port)
    }
  }
  return ports
}
