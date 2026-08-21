// リリース成果物がすべて揃っているかを検証する。
// npm run release の最後に走り、1つでも欠けていれば非0で終了する。
import fs from 'node:fs'
import path from 'node:path'
import { createHash } from 'node:crypto'
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
// SHA256SUMS 用の行を積む (`<hash>  <filename>` = sha256sum -c 互換形式)。
const sha256Lines = []
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
    // SHA-256 を計算して指紋を残す。APK の署名切り替えはしない (利用者判断) が、
    // 配布物のハッシュ一覧があれば改竄検知・再配布時の照合に使える。
    const sum = createHash('sha256').update(fs.readFileSync(file)).digest('hex')
    sha256Lines.push(`${sum}  ${name}`)
    console.log(`  OK   ${name}  ${(size / 1024 / 1024).toFixed(1)}MB`)
}

if (missing.length !== 0) {
    console.error(`\nリリース成果物が ${missing.length} 件欠けています: ${missing.join(', ')}`)
    process.exit(1)
}

// 全件そろったときだけ SHA256SUMS を書き出す。
// `sha256sum -c release/SHA256SUMS_<version>.txt` で検証できる。
const sumsFile = path.join(releaseDir, `SHA256SUMS_${version}.txt`)
fs.writeFileSync(sumsFile, sha256Lines.join('\n') + '\n')
console.log(`\nリリース成果物 ${expected.length} 件すべて揃っています (version ${version})`)
console.log(`SHA-256 一覧を書き出しました: ${sumsFile}`)
