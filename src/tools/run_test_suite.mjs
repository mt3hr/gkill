#!/usr/bin/env node
// 編集前に読む: .claude/skills/gkill-build-test/SKILL.md「リリースゲート（テスト済み attestation）」
//
// テストスイートを 1 本走らせ、成功したら「このツリーでこのスイートが通った」を
// test_attestation.local.json へ記録する。npm run release はこの記録が全スイート分
// 揃っていないと動かない（verify_release_gate.mjs）。
//
// 使い方（package.json の test_* はすべてこれを経由する）:
//   node src/tools/run_test_suite.mjs test_server
//   npm run test_client_e2e -- --workers=2       … 余剰引数は各スイートの実体へ素通しする
//
// 記録するのは次の全部が満たされたときだけ:
//   - スイートが exit 0 で終わった
//   - CI 環境でない（GitHub Actions は CI=true。CI の記録は誰も読まない）
//   - 余剰引数が「網羅性を変えないフラグ」の許可リストに収まっている
//     （並列度・タイムアウト・レポータ形式など）。ファイル名や -run / --grep のような
//     絞り込みは「全件通った」と言えないので記録しない。verify_docs / test_plugins /
//     test_android / test_wear_os は引数が検査内容そのものを置き換える
//     （--list、vet、Gradle タスク名）ので、引数が 1 つでもあれば記録しない
// 記録しない場合も exit は 0 のまま（スイート自体は通っている）。不足はゲートが列挙する。
//
// E2E だけは PATH 上の gkill_server の tree hash も記録する。E2E は PATH のバイナリを
// 起動するので、古いバイナリで通しても現在のツリーを検証したことにならないため。
//
// 依存なし（Node 標準のみ）。

import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import {
  ROOT,
  REQUIRED_SUITES,
  head,
  workingTree,
  recordSuite,
} from './attestation.mjs'

const node = process.execPath
const vitest = [node, path.join(ROOT, 'node_modules', 'vitest', 'vitest.mjs'), 'run']

// 値の形。数値・単純なトークンだけを許し、パスや正規表現が紛れ込まないようにする
const NUMBER = /^\d+$/
const TOKEN = /^[\w.:-]+$/

// 許可リストの書式: { flag, value: 'none' | 'required' | 'optional', re }
const GO_FLAGS = [
  { flag: '-p', value: 'required', re: NUMBER },
  { flag: '-count', value: 'required', re: NUMBER },
  { flag: '-v', value: 'none' },
  { flag: '-timeout', value: 'required', re: TOKEN },
]
const VITEST_FLAGS = [
  { flag: '--pool', value: 'required', re: TOKEN },
  { flag: '--maxWorkers', value: 'required', re: NUMBER },
  { flag: '--no-file-parallelism', value: 'none' },
  { flag: '--reporter', value: 'required', re: TOKEN },
]
const PLAYWRIGHT_FLAGS = [
  { flag: '--workers', value: 'required', re: NUMBER },
  { flag: '--retries', value: 'required', re: NUMBER },
  { flag: '--reporter', value: 'required', re: TOKEN },
]

// Windows では拡張子まで指定しないと spawnSync が実行ファイルを解決できない（test_plugins.mjs と同じ）
const goBin = process.platform === 'win32' ? 'go.exe' : 'go'

export const SUITES = {
  verify_docs: {
    command: [node, path.join(ROOT, 'src', 'tools', 'verify_docs.mjs')],
    cwd: ROOT,
    allowed: [],
  },
  test_server: {
    command: [goBin, 'test', './...'],
    cwd: path.join(ROOT, 'src', 'server'),
    allowed: GO_FLAGS,
  },
  test_client_unit: {
    command: vitest,
    cwd: ROOT,
    allowed: VITEST_FLAGS,
  },
  test_client_e2e: {
    command: [node, path.join(ROOT, 'src', 'client', '__tests__', 'e2e', 'run-e2e.mjs')],
    cwd: ROOT,
    allowed: PLAYWRIGHT_FLAGS,
    recordsServerTree: true,
  },
  // MCP サーバは Go（src/server/gkill/mcp）。test_server の ./... にも含まれるが、
  // スイート名は attestation の互換のため残す（ゲートは test_mcp の記録も要求する）。
  test_mcp: {
    command: [goBin, 'test', './gkill/mcp/...'],
    cwd: path.join(ROOT, 'src', 'server'),
    allowed: GO_FLAGS,
  },
  test_tools: {
    command: [...vitest, '--config', 'vitest.config.tools.ts'],
    cwd: ROOT,
    allowed: VITEST_FLAGS,
  },
  test_plugins: {
    command: [node, path.join(ROOT, 'src', 'tools', 'test_plugins.mjs')],
    cwd: ROOT,
    allowed: [],
  },
  test_android: {
    command: [node, path.join(ROOT, 'src', 'tools', 'gradle_test.mjs'), 'src/android'],
    cwd: ROOT,
    allowed: [],
  },
  test_wear_os: {
    command: [node, path.join(ROOT, 'src', 'tools', 'gradle_test.mjs'), 'src/wear_os'],
    cwd: ROOT,
    allowed: [],
  },
}

// 余剰引数が許可リストに収まっているか。`--workers=2` も `--workers 2` も同じ扱い。
// 戻り値: null（全部許可）か、最初に外れたトークン
export function firstDisallowedArg(args, allowed) {
  for (let i = 0; i < args.length; i++) {
    const token = args[i]
    const eq = token.indexOf('=')
    const flag = eq === -1 ? token : token.slice(0, eq)
    const inline = eq === -1 ? null : token.slice(eq + 1)
    const rule = allowed.find((r) => r.flag === flag)
    if (!rule) return token
    if (rule.value === 'none') {
      if (inline !== null) return token
      continue
    }
    let value = inline
    if (value === null) {
      if (rule.value === 'required') {
        value = args[i + 1]
        if (value === undefined) return token
        i++
      } else {
        continue // optional で値なし
      }
    }
    if (rule.re && !rule.re.test(value)) return `${flag}=${value}`
  }
  return null
}

// `gkill_server version` の出力から tree hash の行を読む。無ければ null
export function parseServerTreeHash(stdout) {
  const m = /^tree:\s+(\S+)\s*$/m.exec(stdout || '')
  return m ? m[1] : null
}

// PATH 上の gkill_server が埋め込んでいる tree hash。run-e2e.mjs と同じホームと --log none で
// 呼び、本番の ~/gkill/logs を触らない（version サブコマンドも PersistentPreRun でログを初期化する）。
export function serverTreeHash() {
  const home = process.env.HOME || process.env.USERPROFILE || ''
  const res = spawnSync('gkill_server', [
    '--gkill_home_dir', path.join(home, 'gkill_test'),
    '--log', 'none',
    'version',
  ], { encoding: 'utf8' })
  if (res.error || res.status !== 0) return null
  return parseServerTreeHash(res.stdout)
}

// 記録するかの判定。スイートの結果と環境と引数から決まり、ここ以外で記録の可否を決めない。
//   { record: true }                      … 記録する
//   { record: false, reason: '...' }      … 記録しない理由（exit には影響しない）
// 「失敗したのに記録する」変更は attestation の前提を壊すので、この関数を単体テストで固定する。
export function decideRecording({ status, signal, ci, args, allowed }) {
  if (signal) return { record: false, reason: `${signal} で中断` }
  if (status !== 0) return { record: false, reason: `exit ${status}` }
  if (ci) return { record: false, reason: 'CI 環境' }
  const bad = firstDisallowedArg(args, allowed)
  if (bad !== null) return { record: false, reason: `引数 ${bad} は全件実行を保証しない` }
  return { record: true }
}

function main() {
  const suite = process.argv[2]
  const args = process.argv.slice(3)
  const spec = SUITES[suite]
  if (!spec) {
    console.error(`未知のスイート: ${suite}（${Object.keys(SUITES).join(' / ')}）`)
    process.exit(2)
  }
  if (!REQUIRED_SUITES.includes(suite)) {
    console.error(`${suite} は REQUIRED_SUITES に無い（attestation.mjs と表を揃えること）`)
    process.exit(2)
  }

  const [cmd, ...cmdArgs] = spec.command
  const res = spawnSync(cmd, [...cmdArgs, ...args], { cwd: spec.cwd, stdio: 'inherit' })
  if (res.error) {
    console.error(`${suite}: ${cmd} を実行できない: ${res.error.message}`)
    process.exit(1)
  }
  if (res.signal) {
    console.error(`${suite}: ${res.signal} で中断`)
    process.exit(1)
  }
  if (res.status !== 0) process.exit(res.status)

  // ここから先はスイートが通ったあとの記録。失敗しても exit 0 のまま（警告だけ出す）
  const decision = decideRecording({ status: res.status, signal: res.signal, ci: Boolean(process.env.CI), args, allowed: spec.allowed })
  if (!decision.record) {
    console.error(`[attestation] ${suite} は記録しない: ${decision.reason}（許可: ${spec.allowed.map((r) => r.flag).join(' ') || 'なし'}）`)
    return
  }
  try {
    const tree = workingTree()
    const entry = {
      tree,
      head: head(),
      finished_at: new Date().toISOString(),
      platform: process.platform,
      args,
    }
    if (spec.recordsServerTree) entry.server_tree = serverTreeHash()
    recordSuite(suite, entry)
    const note = spec.recordsServerTree
      ? `, gkill_server tree=${entry.server_tree ? entry.server_tree.slice(0, 7) : '不明'}`
      : ''
    console.error(`[attestation] ${suite} を記録: tree=${tree.slice(0, 7)}${note}`)
  } catch (e) {
    console.error(`[attestation] ${suite} を記録できない: ${e.message}`)
  }
}

// テストから import できるよう、直接実行されたときだけ走らせる（Windows はドライブ文字の大小が揺れる）
function isDirectRun() {
  if (!process.argv[1]) return false
  const a = path.resolve(process.argv[1])
  const b = fileURLToPath(import.meta.url)
  return process.platform === 'win32' ? a.toLowerCase() === b.toLowerCase() : a === b
}
if (isDirectRun()) main()
