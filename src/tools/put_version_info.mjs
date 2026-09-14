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
import { ROOT, head, workingTree } from './attestation.mjs'

const require = createRequire(import.meta.url)
const pkg = require(path.join(ROOT, 'package.json'))
const version = pkg && pkg.version ? String(pkg.version) : 'unknown'

let commitHash = 'unknown'
try {
  commitHash = head()
} catch {
  commitHash = process.env.GITHUB_SHA || process.env.COMMIT_SHA || 'unknown'
}

let treeHash = 'unknown'
try {
  treeHash = workingTree()
} catch (e) {
  console.error(`[put_version_info] tree_hash を取れない（unknown で続行）: ${e.message}`)
}

// 2026-03-19T10:30:00+09:00 の形（従来のインライン実装と同じ）
const d = new Date()
const pad = (n) => String(n).padStart(2, '0')
const tzMin = -d.getTimezoneOffset()
const sign = tzMin >= 0 ? '+' : '-'
const oh = pad(Math.floor(Math.abs(tzMin) / 60))
const om = pad(Math.abs(tzMin) % 60)
const buildTime = `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}` +
  `T${pad(d.getHours())}:${pad(d.getMinutes())}:${pad(d.getSeconds())}${sign}${oh}:${om}`

const out = path.join(ROOT, 'src', 'server', 'gkill', 'api', 'embed', 'version.json')
fs.mkdirSync(path.dirname(out), { recursive: true })
fs.writeFileSync(out, JSON.stringify({
  commit_hash: commitHash,
  build_time: buildTime,
  version,
  tree_hash: treeHash,
}, null, 2) + '\n')
console.log(`version.json: ${version} commit=${commitHash.slice(0, 7)} tree=${treeHash.slice(0, 7)}`)
