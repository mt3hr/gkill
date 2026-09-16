#!/usr/bin/env node
// 編集前に読む: .claude/skills/gkill-build-test/SKILL.md「リリースゲート（テスト済み attestation）」
//
// src/server/gkill/api/embed/version.json を書く（npm run put_version_info_embed の実体。
// prepare_install の一部として install_server / install_app / release / CI から呼ばれる）。
//
//   commit_hash … HEAD の SHA（git が無ければ GITHUB_SHA / COMMIT_SHA / 'unknown'）
//   build_time  … ローカル時刻（タイムゾーン付き）
//   version     … package.json の version
//   tree_hash   … ビルドした作業ツリーの tree hash（attestation.mjs の workingTree()）。
//                 E2E の attestation はこれを `gkill_server version` から読み取り、
//                 「PATH 上のバイナリが今のツリーから作られている」ことの証拠にする。
//
// ビルドを止めない: git が無い・壊れている環境でも 'unknown' を書いて正常終了する。
// 検証は release ゲート側が行う（unknown のバイナリで E2E を通しても記録が不一致になる）。
//
// 依存なし（Node 標準のみ）。

import fs from 'node:fs'
import path from 'node:path'
import { createRequire } from 'node:module'
import { fileURLToPath } from 'node:url'
import { ROOT, head, workingTree } from './attestation.mjs'

// 2026-03-19T10:30:00+09:00 の形（従来のインライン実装と同じ。ローカル時刻＋タイムゾーン）
export function formatBuildTime(d) {
  const pad = (n) => String(n).padStart(2, '0')
  const tzMin = -d.getTimezoneOffset()
  const sign = tzMin >= 0 ? '+' : '-'
  const oh = pad(Math.floor(Math.abs(tzMin) / 60))
  const om = pad(Math.abs(tzMin) % 60)
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}` +
    `T${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}${sign}${oh}:${om}`
}

// version.json の中身。git が読めないときは 'unknown' で埋める（ビルドを止めない）。
//   readHead / readWorkingTree … 例外を投げうる取得関数（テストで差し替える）
export function buildVersionInfo({
  version,
  now = new Date(),
  env = process.env,
  readHead = head,
  readWorkingTree = workingTree,
  warn = (message) => console.error(message),
} = {}) {
  let commitHash = 'unknown'
  try {
    commitHash = readHead()
  } catch {
    commitHash = env.GITHUB_SHA || env.COMMIT_SHA || 'unknown'
  }
  let treeHash = 'unknown'
  try {
    treeHash = readWorkingTree()
  } catch (e) {
    warn(`[put_version_info] tree_hash を取れない（unknown で続行）: ${e.message}`)
  }
  return {
    commit_hash: commitHash,
    build_time: formatBuildTime(now),
    version: version ? String(version) : 'unknown',
    tree_hash: treeHash,
  }
}

export const VERSION_JSON_PATH = path.join(ROOT, 'src', 'server', 'gkill', 'api', 'embed', 'version.json')

function main() {
  const require = createRequire(import.meta.url)
  const pkg = require(path.join(ROOT, 'package.json'))
  const info = buildVersionInfo({ version: pkg && pkg.version })
  fs.mkdirSync(path.dirname(VERSION_JSON_PATH), { recursive: true })
  fs.writeFileSync(VERSION_JSON_PATH, JSON.stringify(info, null, 2) + '\n')
  console.log(`version.json: ${info.version} commit=${info.commit_hash.slice(0, 7)} tree=${info.tree_hash.slice(0, 7)}`)
}

// テストから import できるよう、直接実行されたときだけ走らせる（Windows はドライブ文字の大小が揺れる）
function isDirectRun() {
  if (!process.argv[1]) return false
  const a = path.resolve(process.argv[1])
  const b = fileURLToPath(import.meta.url)
  return process.platform === 'win32' ? a.toLowerCase() === b.toLowerCase() : a === b
}
if (isDirectRun()) main()
