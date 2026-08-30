// リリース成果物がすべて揃っているかを検証する。
// npm run release の最後に走り、1つでも欠けていれば非0で終了する。
import fs from 'node:fs'
import path from 'node:path'
import { createHash } from 'node:crypto'
import { createRequire } from 'node:module'
import { execFileSync } from 'node:child_process'

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
    // SHA-256 を計算して指紋を残す。
    // 配布物のハッシュ一覧があれば改竄検知・再配布時の照合に使える。
    const sum = createHash('sha256').update(fs.readFileSync(file)).digest('hex')
    sha256Lines.push(`${sum}  ${name}`)
    console.log(`  OK   ${name}  ${(size / 1024 / 1024).toFixed(1)}MB`)
}

if (missing.length !== 0) {
    console.error(`\nリリース成果物が ${missing.length} 件欠けています: ${missing.join(', ')}`)
    process.exit(1)
}

// サンプルデータ zip は「存在してサイズがある」だけでは足りない。
// prepare_gkill_sample_data のコピー・同梱手順が崩れると、bat や exe や DB を
// 欠いたまま正常サイズの zip ができてしまう（利用者が起動して初めて気付く）。
// パックと同じ 7za で一覧を取り、必須エントリの実在を検査する。
const sampleZip = path.join(releaseDir, `gkill_sample_data_${version}.zip`)
const requiredSampleEntries = [
    'gkill_sample_data/LAUNCH_GKILL_SAMPLE_DATA.bat',
    'gkill_sample_data/README.txt',
    'gkill_sample_data/gkill_server.exe',
    'gkill_sample_data/configs/account.db',
    'gkill_sample_data/configs/user_config.db',
    'gkill_sample_data/configs/server_config.db',
    'gkill_sample_data/datas/gkill_sample_data/Kmemo.db',
    'gkill_sample_data/datas/gkill_sample_data/Files',
    'gkill_sample_data/datas/gkill_sample_data/GPSLog',
]
let sampleZipListing = ''
try {
    // -slt: 1エントリ = 「Path = <パス>」行になる機械可読形式
    sampleZipListing = execFileSync('7za', ['l', '-slt', sampleZip], { encoding: 'utf8' })
} catch (e) {
    console.error(`\ngkill_sample_data zip の一覧取得に失敗しました (7za l): ${e.message}`)
    process.exit(1)
}
const sampleZipPaths = new Set(
    sampleZipListing
        .split('\n')
        .filter((line) => line.startsWith('Path = '))
        // 7za の Path 区切りは環境で \ になりうるので / へ揃える
        .map((line) => line.slice('Path = '.length).trim().replaceAll('\\', '/')),
)
const missingSampleEntries = requiredSampleEntries.filter((entry) => !sampleZipPaths.has(entry))
if (missingSampleEntries.length !== 0) {
    console.error(`\ngkill_sample_data zip に必須エントリが ${missingSampleEntries.length} 件ありません: ${missingSampleEntries.join(', ')}`)
    process.exit(1)
}
console.log(`  OK   gkill_sample_data zip の必須エントリ ${requiredSampleEntries.length} 件を確認`)

// APK 3件の署名検証。以前は assembleDebug の成果物を配布名へ rename しており、
// debug 鍵で署名された APK が正式版として公開されていた (2026-08-30 監査 F-006)。
// apksigner verify が通り、かつ署名者が Android の debug 証明書 (CN=Android Debug)
// でないことを確認する。apksigner が見つからない場合は fail-closed で止める
// (黙って未検証のまま配布物を作らない)。
function findApksigner() {
    const sdkRoot = process.env.ANDROID_HOME || process.env.ANDROID_SDK_ROOT
    if (!sdkRoot) return null
    const buildToolsDir = path.join(sdkRoot, 'build-tools')
    let versions = []
    try {
        versions = fs.readdirSync(buildToolsDir).sort().reverse()
    } catch {
        return null
    }
    for (const v of versions) {
        for (const bin of ['apksigner', 'apksigner.bat']) {
            const candidate = path.join(buildToolsDir, v, bin)
            if (fs.existsSync(candidate)) return candidate
        }
    }
    return null
}

const apkNames = expected.filter((name) => name.endsWith('.apk'))
const apksigner = findApksigner()
if (apksigner === null) {
    console.error('\napksigner が見つかりません (ANDROID_HOME / ANDROID_SDK_ROOT の build-tools 配下)。')
    console.error('APK の署名検証ができないため release を中止します。')
    process.exit(1)
}
for (const name of apkNames) {
    const file = path.join(releaseDir, name)
    let certsOut = ''
    try {
        certsOut = execFileSync(apksigner, ['verify', '--print-certs', file], { encoding: 'utf8' })
    } catch (e) {
        console.error(`\n  NG   ${name} の署名検証に失敗しました (未署名または署名破損): ${e.message}`)
        process.exit(1)
    }
    if (certsOut.includes('CN=Android Debug')) {
        console.error(`\n  NG   ${name} が Android の debug 鍵で署名されています。リリース鍵で署名し直すこと`)
        process.exit(1)
    }
    const fingerprint = certsOut
        .split('\n')
        .find((line) => line.includes('certificate SHA-256 digest'))
    console.log(`  OK   ${name} 署名検証済み${fingerprint ? ` (${fingerprint.trim()})` : ''}`)
}

// 全件そろったときだけ SHA256SUMS を書き出す。
// `sha256sum -c release/SHA256SUMS_<version>.txt` で検証できる。
const sumsFile = path.join(releaseDir, `SHA256SUMS_${version}.txt`)
fs.writeFileSync(sumsFile, sha256Lines.join('\n') + '\n')
console.log(`\nリリース成果物 ${expected.length} 件すべて揃っています (version ${version})`)
console.log(`SHA-256 一覧を書き出しました: ${sumsFile}`)
