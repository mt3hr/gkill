// リリース成果物がすべて揃っているかを検証する。
// npm run release の最後に走り、1つでも欠けていれば非0で終了する。
//
// 判定（成果物の一覧・7za の一覧の読み方・必須エントリ・debug 署名の判定・apksigner の探索）は
// 関数に切り出してあり、src/tools/__tests__/release_scripts.test.mjs が固定する。
// main() は直接実行されたときだけ走る（テストから import しても何もしない）。
import fs from 'node:fs'
import path from 'node:path'
import { createHash } from 'node:crypto'
import { createRequire } from 'node:module'
import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { head, headTree, dirtyEntries, releaseAttestationName } from './attestation.mjs'

// リリース成果物の一覧。先頭はリリースゲート（verify_release_gate.mjs）が release の先頭で
// 書いた記録で、成果物として SHA256SUMS に載せ、ビルド中に HEAD が動いていないことも検査する
export function expectedArtifacts(version) {
    return [
        releaseAttestationName(version),
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
}

// サンプルデータ zip は「存在してサイズがある」だけでは足りない。
// prepare_gkill_sample_data のコピー・同梱手順が崩れると、bat や exe や DB を
// 欠いたまま正常サイズの zip ができてしまう（利用者が起動して初めて気付く）。
export const REQUIRED_SAMPLE_ENTRIES = [
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

// `7za l -slt` の出力（1エントリ = 「Path = <パス>」行）からパスの集合を作る。
// 7za の Path 区切りは環境で \ になりうるので / へ揃える
export function parse7zaListingPaths(listing) {
    return new Set(
        String(listing)
            .split('\n')
            .filter((line) => line.startsWith('Path = '))
            .map((line) => line.slice('Path = '.length).trim().replaceAll('\\', '/')),
    )
}

export function missingSampleEntries(paths, required = REQUIRED_SAMPLE_ENTRIES) {
    return required.filter((entry) => !paths.has(entry))
}

// APK が Android の debug 証明書で署名されているか（apksigner verify --print-certs の出力から）。
// 以前は assembleDebug の成果物を配布名へ rename しており、debug 鍵署名の APK が正式版として
// 公開されていた (指摘 F-006)
export function isDebugSigned(certsOut) {
    return String(certsOut).includes('CN=Android Debug')
}

export function certificateFingerprintLine(certsOut) {
    const line = String(certsOut)
        .split('\n')
        .find((candidate) => candidate.includes('certificate SHA-256 digest'))
    return line ? line.trim() : null
}

// ANDROID_HOME / ANDROID_SDK_ROOT の build-tools 配下から最新の apksigner を探す。無ければ null
export function findApksigner(env = process.env, fsImpl = fs) {
    const sdkRoot = env.ANDROID_HOME || env.ANDROID_SDK_ROOT
    if (!sdkRoot) return null
    const buildToolsDir = path.join(sdkRoot, 'build-tools')
    let versions = []
    try {
        versions = fsImpl.readdirSync(buildToolsDir).sort().reverse()
    } catch {
        return null
    }
    for (const v of versions) {
        for (const bin of ['apksigner', 'apksigner.bat']) {
            const candidate = path.join(buildToolsDir, v, bin)
            if (fsImpl.existsSync(candidate)) return candidate
        }
    }
    return null
}

function main() {
    const require = createRequire(import.meta.url)
    const version = require('../../package.json').version
    const releaseDir = 'release'
    const expected = expectedArtifacts(version)

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

    // パックと同じ 7za で一覧を取り、必須エントリの実在を検査する。
    const sampleZip = path.join(releaseDir, `gkill_sample_data_${version}.zip`)
    let sampleZipListing = ''
    try {
        // -slt: 1エントリ = 「Path = <パス>」行になる機械可読形式
        sampleZipListing = execFileSync('7za', ['l', '-slt', sampleZip], { encoding: 'utf8' })
    } catch (e) {
        console.error(`\ngkill_sample_data zip の一覧取得に失敗しました (7za l): ${e.message}`)
        process.exit(1)
    }
    const missingEntries = missingSampleEntries(parse7zaListingPaths(sampleZipListing))
    if (missingEntries.length !== 0) {
        console.error(`\ngkill_sample_data zip に必須エントリが ${missingEntries.length} 件ありません: ${missingEntries.join(', ')}`)
        process.exit(1)
    }
    console.log(`  OK   gkill_sample_data zip の必須エントリ ${REQUIRED_SAMPLE_ENTRIES.length} 件を確認`)

    // APK 3件の署名検証。apksigner verify が通り、かつ署名者が Android の debug 証明書
    // (CN=Android Debug) でないことを確認する。apksigner が見つからない場合は fail-closed で止める
    // (黙って未検証のまま配布物を作らない)。
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
        if (isDebugSigned(certsOut)) {
            console.error(`\n  NG   ${name} が Android の debug 鍵で署名されています。リリース鍵で署名し直すこと`)
            process.exit(1)
        }
        const fingerprint = certificateFingerprintLine(certsOut)
        console.log(`  OK   ${name} 署名検証済み${fingerprint ? ` (${fingerprint})` : ''}`)
    }

    // リリースゲートの記録が「今ビルドしたもの」を指しているかを検査する。
    // ゲートは release の先頭で通るが、prepare_install 〜 APK ビルドの 30 分の間に
    // コミットや編集が入っても Go / Gradle は黙ってそれを取り込む。ゲート時の HEAD と
    // tree に一致し、かつ今も作業ツリーがクリーンでなければ成果物とは認めない。
    const attestation = JSON.parse(fs.readFileSync(path.join(releaseDir, releaseAttestationName(version)), 'utf8'))
    const nowHead = head()
    const nowTree = headTree()
    if (attestation.head !== nowHead || attestation.tree !== nowTree) {
        console.error(`\n  NG   リリースゲート通過後に HEAD が動いている (gate ${String(attestation.head).slice(0, 7)} → now ${nowHead.slice(0, 7)})。`)
        console.error('       npm run release を最初からやり直すこと')
        process.exit(1)
    }
    const dirty = dirtyEntries()
    if (dirty.length !== 0) {
        console.error(`\n  NG   ビルド中に作業ツリーが変わっている (${dirty.length} 件: ${dirty.slice(0, 5).join(' / ')})。`)
        console.error('       npm run release を最初からやり直すこと')
        process.exit(1)
    }
    console.log(`  OK   リリースゲートの記録と HEAD ${nowHead.slice(0, 7)} / tree ${nowTree.slice(0, 7)} が一致、作業ツリーはクリーン`)

    // 全件そろったときだけ SHA256SUMS を書き出す。
    // `sha256sum -c release/SHA256SUMS_<version>.txt` で検証できる。
    const sumsFile = path.join(releaseDir, `SHA256SUMS_${version}.txt`)
    fs.writeFileSync(sumsFile, sha256Lines.join('\n') + '\n')
    console.log(`\nリリース成果物 ${expected.length} 件すべて揃っています (version ${version})`)
    console.log(`SHA-256 一覧を書き出しました: ${sumsFile}`)
}

// テストから import できるよう、直接実行されたときだけ走らせる（Windows はドライブ文字の大小が揺れる）
function isDirectRun() {
    if (!process.argv[1]) return false
    const a = path.resolve(process.argv[1])
    const b = fileURLToPath(import.meta.url)
    return process.platform === 'win32' ? a.toLowerCase() === b.toLowerCase() : a === b
}
if (isDirectRun()) main()
