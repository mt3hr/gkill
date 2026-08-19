#!/usr/bin/env node
// Android / Wear OS の Gradle テストを走らせる。
//
// `bash gradlew test` を直接書いていたが、Windows では `bash` が WSL の bash に解決され、
// WSL 側から Windows の JDK / Android SDK が見えずに落ちる。
// ビルド側（build_android_apk / build_wear_os_*）は
// `process.platform === 'win32' ? 'gradlew.bat' : './gradlew'` と分岐しており、
// テスト側だけがその分岐から漏れていた。ここで同じ形へ揃える。
//
// 使い方:
//   node src/tools/gradle_test.mjs src/android
//   node src/tools/gradle_test.mjs src/wear_os
//
// 依存なし（Node 標準のみ）。

import fs from 'node:fs'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..', '..')

const target = process.argv[2]
if (!target) {
  console.error('プロジェクトのパスを指定すること (例: src/android)')
  process.exit(1)
}
const projectDir = path.resolve(ROOT, target)
if (!fs.existsSync(projectDir)) {
  console.error(`プロジェクトが見つからない: ${target}`)
  process.exit(1)
}

// gradlew は CRLF で取り込まれていると sh が読めないので LF へ揃える（ビルド側と同じ手当て）
const wrapper = path.join(projectDir, 'gradlew')
if (fs.existsSync(wrapper)) {
  fs.writeFileSync(wrapper, fs.readFileSync(wrapper, 'utf8').replace(/\r\n/g, '\n'))
  try {
    fs.chmodSync(wrapper, 0o755)
  } catch {
    // Windows では chmod は効かないが、gradlew.bat を使うので問題ない
  }
}

const command = process.platform === 'win32' ? 'gradlew.bat' : './gradlew'
const args = process.argv.slice(3)
const res = spawnSync(command, args.length > 0 ? args : ['test'], {
  cwd: projectDir,
  stdio: 'inherit',
  shell: process.platform === 'win32',
})
if (res.error) {
  console.error(`${command} を実行できない: ${res.error.message}`)
  process.exit(1)
}
process.exit(res.status ?? 1)
