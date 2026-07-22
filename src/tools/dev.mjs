/**
 * Vite dev server launcher.
 *
 * `npm run dev -- --api=<url>` で接続先のgkill_serverを指定できるようにするラッパー。
 * Viteは未知のCLIオプションをエラーにするため、--api系はここで取り除いてから
 * vite本体に残りの引数を渡す。接続先は GKILL_API_PROXY_TARGET 環境変数として
 * 渡し、vite.config.ts のproxy設定がそれを読む (E2Eのrun-e2e.mjsと同じ経路)。
 */
import { spawn } from 'node:child_process'
import path from 'node:path'

const DEFAULT_TARGET = 'http://localhost:9999'
const API_OPTIONS = ['--api', '--api-target']

/**
 * 接続先の書き味を優先して補完する。
 *   19999           -> http://127.0.0.1:19999
 *   :19999          -> http://127.0.0.1:19999
 *   example.com     -> http://example.com
 *   https://host    -> そのまま
 */
function normalizeTarget(value) {
  const raw = String(value ?? '').trim()
  if (raw === '') {
    return null
  }

  let url = raw
  if (/^\d+$/.test(url)) {
    url = `http://127.0.0.1:${url}`
  } else if (url.startsWith(':')) {
    url = `http://127.0.0.1${url}`
  } else if (!/^https?:\/\//.test(url)) {
    url = `http://${url}`
  }

  try {
    const parsed = new URL(url)
    if (!parsed.hostname) {
      return null
    }
    return parsed.origin
  } catch {
    return null
  }
}

// --api=<value> / --api <value> を取り除きつつ値を拾う。最後に指定されたものを採用する
const argv = process.argv.slice(2)
const viteArgs = []
let apiArg = null

for (let i = 0; i < argv.length; i++) {
  const arg = argv[i]
  const matched = API_OPTIONS.find((name) => arg === name || arg.startsWith(`${name}=`))
  if (!matched) {
    viteArgs.push(arg)
    continue
  }
  if (arg === matched) {
    apiArg = argv[++i] ?? ''
  } else {
    apiArg = arg.slice(matched.length + 1)
  }
}

let target = process.env.GKILL_API_PROXY_TARGET || DEFAULT_TARGET
if (apiArg !== null) {
  const normalized = normalizeTarget(apiArg)
  if (!normalized) {
    console.error(`[dev] 接続先として解釈できません: "${apiArg}"`)
    console.error('[dev] 例: --api=http://127.0.0.1:19999 / --api=19999 / --api=example.com:9999')
    process.exit(1)
  }
  target = normalized
}

console.log(`[dev] API proxy target: ${target}`)

// シェル経由 (npx) だとWindowsでkillが子プロセスに届かないのでnodeで直接起動する
const viteBin = path.join(process.cwd(), 'node_modules', 'vite', 'bin', 'vite.js')
const vite = spawn(process.execPath, [viteBin, ...viteArgs], {
  stdio: 'inherit',
  env: { ...process.env, GKILL_API_PROXY_TARGET: target },
})

for (const signal of ['SIGINT', 'SIGTERM']) {
  process.on(signal, () => {
    if (vite.exitCode === null) {
      vite.kill(signal)
    }
  })
}

vite.on('error', (e) => {
  console.error(`[dev] Viteの起動に失敗しました: ${e.message}`)
  process.exit(1)
})

vite.on('exit', (code, signal) => {
  process.exit(code ?? (signal ? 1 : 0))
})
