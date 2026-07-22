// リリース成果物がすべて揃っているかを検証する。
// npm run release の最後に走り、1つでも欠けていれば非0で終了する。
import fs from 'node:fs'
import path from 'node:path'
import { createRequire } from 'node:module'

const require = createRequire(import.meta.url)
const version = require('../../package.json').version
const releaseDir = 'release'

const expected = [
    `windows_amd64_gkill_${version}.zip`,
    `windows_amd64_gkill_server_${version}.zip`,
    `linux_amd64_gkill_server_${version}.zip`,
    `linux_arm64_gkill_server_${version}.zip`,
    `linux_arm_gkill_server_${version}.zip`,
    `android_arm_gkill_server_${version}.zip`,
    `android_arm64_gkill_server_${version}.zip`,
    `gkill_${version}.apk`,
    `gkill_wear_companion_${version}.apk`,
    `gkill_wear_watch_${version}.apk`,
    `gkill_sample_data_${version}.zip`,
]

const missing = []
for (const name of expected) {
    const file = path.join(releaseDir, name)
    let size = 0
    try {
        size = fs.statSync(file).size
    } catch {
        missing.push(name)
        console.error(`  NG   ${name} (存在しません)`)
        continue
    }
    if (size === 0) {
        missing.push(name)
        console.error(`  NG   ${name} (サイズ0)`)
        continue
    }
    console.log(`  OK   ${name}  ${(size / 1024 / 1024).toFixed(1)}MB`)
}

if (missing.length !== 0) {
    console.error(`\nリリース成果物が ${missing.length} 件欠けています: ${missing.join(', ')}`)
    process.exit(1)
}
console.log(`\nリリース成果物 ${expected.length} 件すべて揃っています (version ${version})`)
