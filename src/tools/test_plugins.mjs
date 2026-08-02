#!/usr/bin/env node
// src/plugins/ 配下の各プラグインのGoテストを走らせる。
//
// 各プラグインは独立した go.mod を持つ別モジュールなので、
// `cd src/server && go test ./...`（npm run test_server）では実行されない。
// モジュールを1つずつ回すためだけのスクリプト。
//
// 使い方: npm run test_plugins
//
// 依存なし（Node 標準のみ）。

import fs from 'node:fs'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..', '..') // src/tools/ → リポジトリルート
const PLUGINS_DIR = path.join(ROOT, 'src', 'plugins')

// go.mod を持つディレクトリ = 独立モジュール。examples/ 配下も拾えるよう再帰する。
function findModules(dir) {
  const out = []
  if (!fs.existsSync(dir)) return out
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue
    const child = path.join(dir, entry.name)
    if (fs.existsSync(path.join(child, 'go.mod'))) {
      out.push(child)
      continue // モジュール内はそのモジュールの go test ./... が面倒を見る
    }
    out.push(...findModules(child))
  }
  return out
}

const modules = findModules(PLUGINS_DIR).sort()
if (modules.length === 0) {
  console.error('プラグインモジュールが見つからない: src/plugins/')
  process.exit(1)
}

const failed = []
for (const mod of modules) {
  const rel = path.relative(ROOT, mod).split(path.sep).join('/')
  console.log(`\n=== go test ./...  (${rel})`)
  // shell は使わない（args がエスケープされない旨の DeprecationWarning を避ける）。
  // Windows では拡張子まで指定しないと spawnSync が実行ファイルを解決できない。
  const goBin = process.platform === 'win32' ? 'go.exe' : 'go'
  const res = spawnSync(goBin, ['test', './...'], { cwd: mod, stdio: 'inherit' })
  if (res.error) {
    console.error(`  ${goBin} を実行できない: ${res.error.message}`)
    failed.push(rel)
    continue
  }
  if (res.status !== 0) failed.push(rel)
}

if (failed.length > 0) {
  console.error(`\n❌ 失敗したモジュール ${failed.length}件:`)
  for (const f of failed) console.error('  - ' + f)
  process.exit(1)
}
console.log(`\n✅ プラグイン ${modules.length}モジュール すべて成功`)
