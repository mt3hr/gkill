#!/usr/bin/env node
// CodeQL をローカルで回す。
//
// CI（.github/workflows/codeql.yml）と同じ設定ファイル・同じクエリスイートを使い、
// push する前に GitHub の Code scanning アラートを手元で再現するためのもの。
//
// **設定ファイルは新しく作らない。** 解析対象パス（paths-ignore）は
// .github/codeql/codeql-config.yml を CI と共用する。ローカル専用の設定を別に置くと
// CI とドリフトして「ローカルでは消えたのに CI では出る」という一番タチの悪い状態になる。
//
// 使い方:
//   npm run codeql                       … go + javascript-typescript
//   npm run codeql -- go                 … 言語を指定（go / javascript-typescript / java-kotlin）
//   npm run codeql -- --all              … java-kotlin も含める（Gradle ビルドを伴う。後述）
//   npm run codeql -- --update-baseline  … 既知の指摘一覧を今回の結果で作り直す
//   npm run codeql -- --reuse-db         … 既存のデータベースを作り直さない（クエリだけ回し直す）
//   npm run codeql -- --required         … CodeQL CLI が無いときスキップではなく失敗にする
//
// CodeQL CLI が無い環境では**スキップして正常終了する**ので、npm test に混ぜても壊れない。
//
// 環境変数:
//   GKILL_CODEQL       … CodeQL CLI の実行ファイル、またはバンドルを展開したディレクトリ
//   GKILL_CODEQL_WORK  … データベース・SARIF の置き場所（既定は OS のテンポラリ配下）
//
// 成果物はリポジトリの外に置く。言語ごとに数GBになるので、
// 空きのあるドライブを GKILL_CODEQL_WORK で指定すること。
//
// 依存なし（Node 標準のみ）。

import fs from 'node:fs'
import os from 'node:os'
import path from 'node:path'
import { spawnSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..', '..') // src/tools/ → リポジトリルート
const CONFIG_FILE = path.join(ROOT, '.github', 'codeql', 'codeql-config.yml')
const BASELINE_FILE = path.join(ROOT, '.github', 'codeql', 'local-baseline.json')

// CI の matrix と同じ3言語。
// クエリパック名は言語IDと綴りが違う（java-kotlin → java）ので対応表を持つ。
const LANGUAGES = {
  'go': { pack: 'go', buildMode: 'autobuild' },
  'javascript-typescript': { pack: 'javascript', buildMode: 'none' },
  'java-kotlin': { pack: 'java', buildMode: 'manual' },
}
// 既定は「ワークツリーを汚さず、Gradle も要らない」2言語だけ。
// java-kotlin は追跡ファイルの書き換えと重い Gradle ビルドを伴うので明示指定させる。
const DEFAULT_LANGUAGES = ['go', 'javascript-typescript']

// java-kotlin の解析にだけ必要な Kotlin の固定（CI の "Pin Kotlin version for CodeQL" と同じ）。
// CodeQL の Kotlin extractor は 2.4.10 未満にしか対応しておらず、
// 2.4.10 のままだと compileDebugKotlin が KotlinVersionTooRecentError で落ちて解析が0件になる。
const KOTLIN_PIN_VERSION = '2.4.0'
const KOTLIN_PIN_FILES = ['src/android/gradle/libs.versions.toml', 'src/wear_os/gradle/libs.versions.toml']

// --no-build-cache が要る理由は codeql.yml のコメントと同じ。
// ビルドキャッシュが効くとコンパイルタスクが実際には走らず、CodeQL が何も観測できない。
const GRADLE_BUILDS = [
  { dir: 'src/android', tasks: [':app:assembleDebug'] },
  { dir: 'src/wear_os', tasks: [':watch_app:assembleDebug', ':phone_companion:assembleDebug'] },
]
const GRADLE_FLAGS = ['--no-build-cache', '--no-daemon', '--stacktrace']

const IS_WIN = process.platform === 'win32'
const KNOWN_FLAGS = ['--all', '--required', '--update-baseline', '--reuse-db']

// ── 引数 ───────────────────────────────────────────────────────────────

const argv = process.argv.slice(2)
const flags = new Set(argv.filter((a) => a.startsWith('--')))
const requested = argv.filter((a) => !a.startsWith('--'))

const unknownFlag = [...flags].find((f) => !KNOWN_FLAGS.includes(f))
if (unknownFlag) {
  console.error(`未対応のオプション: ${unknownFlag}（${KNOWN_FLAGS.join(' / ')} のみ）`)
  process.exit(1)
}

const unknownLang = requested.find((l) => !(l in LANGUAGES))
if (unknownLang) {
  console.error(`未対応の言語: ${unknownLang}（${Object.keys(LANGUAGES).join(' / ')} のみ）`)
  process.exit(1)
}

const languages =
  requested.length > 0 ? requested : flags.has('--all') ? Object.keys(LANGUAGES) : DEFAULT_LANGUAGES

// ── 小道具 ─────────────────────────────────────────────────────────────

function isFile(p) {
  try {
    return fs.statSync(p).isFile()
  } catch {
    return false
  }
}

// shell は使わない（引数がエスケープされない旨の DeprecationWarning を避ける）。
// Windows では実行ファイルを拡張子まで指定して解決させる。
function run(command, args, opts = {}) {
  const res = spawnSync(command, args, { stdio: 'inherit', cwd: ROOT, ...opts })
  if (res.error) {
    console.error(`  ${command} を実行できない: ${res.error.message}`)
    return 1
  }
  return res.status ?? 1
}

// ── CodeQL CLI の探索 ──────────────────────────────────────────────────

function findCodeqlCli() {
  const exe = IS_WIN ? 'codeql.exe' : 'codeql'

  // 1. 明示指定。実行ファイルそのものでも、バンドルを展開したディレクトリでもよい。
  //    外れているときに PATH へ落とさないのは、指定したつもりの版と違うものを
  //    黙って使ってしまうのを防ぐため。
  const specified = process.env.GKILL_CODEQL
  if (specified) {
    for (const candidate of [specified, path.join(specified, exe), path.join(specified, 'codeql', exe)]) {
      if (isFile(candidate)) return candidate
    }
    return null
  }

  // 2. PATH
  for (const dir of (process.env.PATH || '').split(path.delimiter).filter(Boolean)) {
    const candidate = path.join(dir, exe)
    if (isFile(candidate)) return candidate
  }

  // 3. よくある展開先。バンドルは codeql/ というディレクトリを作って展開されるので両方見る。
  const home = os.homedir()
  const roots = IS_WIN
    ? ['D:\\tools\\codeql', 'C:\\tools\\codeql', path.join(home, 'codeql'), process.env.LOCALAPPDATA]
    : ['/opt/codeql', '/usr/local/codeql', path.join(home, 'codeql')]
  for (const root of roots.filter(Boolean)) {
    for (const candidate of [path.join(root, exe), path.join(root, 'codeql', exe)]) {
      if (isFile(candidate)) return candidate
    }
  }
  return null
}

function printInstallInstructions() {
  const bundle = IS_WIN ? 'codeql-bundle-win64.tar.gz' : 'codeql-bundle-linux64.tar.gz'
  console.log('')
  console.log('   導入する場合（gkill は public リポジトリなので CodeQL CLI のライセンス制約に当たらない）:')
  console.log('     1. https://github.com/github/codeql-action/releases から')
  console.log(`        ${bundle} を取得して展開する`)
  console.log('        （クエリ同梱版。CLI 単体だとクエリパックを別途取りに行くことになる）')
  console.log('     2. 展開先を GKILL_CODEQL に設定するか、codeql を PATH に通す')
  console.log('     3. データベースは数GBになるので GKILL_CODEQL_WORK で空きのある場所を指定する')
  console.log('')
}

// ── Kotlin のバージョン固定（java-kotlin のときだけ） ──────────────────

// CI は使い捨ての作業ツリーなので sed -i でよいが、ローカルでは追跡ファイルを書き換えることになる。
// 元の内容を控えて finally で必ず戻す。git checkout は使わない（HEAD に戻ってしまい、
// 関係のない未コミット変更まで消える）。
function withKotlinPin(fn) {
  const gitBin = IS_WIN ? 'git.exe' : 'git'
  const status = spawnSync(gitBin, ['status', '--porcelain', '--', ...KOTLIN_PIN_FILES], {
    cwd: ROOT,
    encoding: 'utf8',
  })
  if (status.status === 0 && status.stdout.trim() !== '') {
    console.error('❌ Kotlin のバージョンファイルに未コミットの変更がある。')
    console.error('   解析中に一時的に書き換えるので、巻き込まないようコミットするか退避すること:')
    for (const rel of KOTLIN_PIN_FILES) console.error(`     ${rel}`)
    return 1
  }

  const originals = new Map()
  for (const rel of KOTLIN_PIN_FILES) {
    const abs = path.join(ROOT, rel)
    originals.set(abs, fs.readFileSync(abs, 'utf8'))
  }

  const restore = () => {
    for (const [abs, body] of originals) {
      try {
        fs.writeFileSync(abs, body)
      } catch (e) {
        console.error(`⚠ 元に戻せなかった: ${abs} (${e.message})`)
      }
    }
  }
  const onSignal = () => {
    restore()
    process.exit(130)
  }
  process.on('SIGINT', onSignal)
  process.on('SIGTERM', onSignal)

  try {
    for (const [abs, body] of originals) {
      // 作業ツリーは CRLF なので、行末を当てにする置換は書かない（黙って0件になる）。
      // 置換できたかを必ず数えて、当たらなければそこで止める。
      let hits = 0
      const pinned = body.replace(/^kotlin = "[^"]*"/gm, () => {
        hits += 1
        return `kotlin = "${KOTLIN_PIN_VERSION}"`
      })
      if (hits === 0) {
        console.error(`❌ kotlin のバージョン行が見つからない: ${path.relative(ROOT, abs)}`)
        console.error('   CI の "Pin Kotlin version for CodeQL" と同じ置換が当たらない。書式が変わった可能性がある。')
        return 1
      }
      fs.writeFileSync(abs, pinned)
    }
    console.log(`ℹ Kotlin を ${KOTLIN_PIN_VERSION} に固定した（解析中のみ。終了時に元へ戻す）`)
    return fn()
  } finally {
    restore()
    process.off('SIGINT', onSignal)
    process.off('SIGTERM', onSignal)
  }
}

// ── データベース作成・解析 ─────────────────────────────────────────────

function gradleCommandFor(build) {
  const dir = path.join(ROOT, build.dir)
  const tasks = [...build.tasks, ...GRADLE_FLAGS].join(' ')
  // --command は CodeQL 側でシェルに渡されるので、作業ディレクトリの移動もここに書く。
  // ラッパーは絶対パスで呼ぶ（裸の名前だと NoDefaultCurrentDirectoryInExePath=1 の環境で
  // カレントディレクトリを探索せず「認識されていません」で落ちる）。
  return IS_WIN
    ? `cd /d "${dir}" && "${path.join(dir, 'gradlew.bat')}" ${tasks}`
    : `cd "${dir}" && "${path.join(dir, 'gradlew')}" ${tasks}`
}

function createDatabase(cli, lang, dbDir) {
  const args = [
    'database',
    'create',
    dbDir,
    `--language=${lang}`,
    `--source-root=${ROOT}`,
    `--codescanning-config=${CONFIG_FILE}`,
    '--overwrite',
  ]
  if (LANGUAGES[lang].buildMode === 'manual') {
    for (const build of GRADLE_BUILDS) args.push(`--command=${gradleCommandFor(build)}`)
  } else {
    args.push(`--build-mode=${LANGUAGES[lang].buildMode}`)
  }
  console.log(`\n=== codeql database create (${lang})`)
  return run(cli, args)
}

function analyzeDatabase(cli, lang, dbDir, sarifPath) {
  // CI の config は `queries: - uses: security-and-quality`。
  // CLI では言語ごとのスイートを明示的に渡す（config の queries: は CLI 側では解決されない）。
  const pack = LANGUAGES[lang].pack
  const suite = `codeql/${pack}-queries:codeql-suites/${pack}-security-and-quality.qls`
  console.log(`\n=== codeql database analyze (${lang})`)
  return run(cli, [
    'database',
    'analyze',
    dbDir,
    suite,
    '--format=sarif-latest',
    `--output=${sarifPath}`,
    `--sarif-category=/language:${lang}`,
    '--threads=0',
  ])
}

// ── SARIF の読み取り ───────────────────────────────────────────────────

// 行番号は無関係な編集でも動くので、指標には CodeQL が出す primaryLocationLineHash を使う。
// 無い場合だけ行番号へ落とす。
function fingerprintOf(result, ruleId, file) {
  const lineHash = result.partialFingerprints?.primaryLocationLineHash
  if (lineHash) return `${ruleId}|${file}|${lineHash}`
  const line = result.locations?.[0]?.physicalLocation?.region?.startLine ?? 0
  return `${ruleId}|${file}|L${line}`
}

function readSarif(sarifPath, lang) {
  const sarif = JSON.parse(fs.readFileSync(sarifPath, 'utf8'))
  const findings = []
  for (const sarifRun of sarif.runs ?? []) {
    const rules = sarifRun.tool?.driver?.rules ?? []
    for (const result of sarifRun.results ?? []) {
      const rule = result.rule?.index !== undefined ? rules[result.rule.index] : null
      const ruleId = result.ruleId ?? rule?.id ?? '(unknown rule)'
      const location = result.locations?.[0]?.physicalLocation
      const file = location?.artifactLocation?.uri ?? '(no location)'
      findings.push({
        fingerprint: fingerprintOf(result, ruleId, file),
        lang,
        rule: ruleId,
        severity: rule?.properties?.['problem.severity'] ?? result.level ?? 'unknown',
        file,
        line: location?.region?.startLine ?? 0,
        message: (result.message?.text ?? '').split('\n')[0],
      })
    }
  }
  return findings
}

// ── ベースライン ───────────────────────────────────────────────────────

function loadBaseline() {
  if (!isFile(BASELINE_FILE)) return null
  const parsed = JSON.parse(fs.readFileSync(BASELINE_FILE, 'utf8'))
  return parsed.findings ?? []
}

// 実行するたびに diff が出ないよう、時刻のような揺れる値は入れず指標順に並べて書く。
//
// **今回走らせた言語ぶんだけを差し替える。** `npm run codeql -- go --update-baseline` で
// 全体を書き直すと、走らせていない javascript-typescript / java-kotlin の記録が
// 黙って消え、次に全言語で回したとき「新しい指摘」として一斉に赤くなる。
// 言語が判別できない古い記録（lang 無し）は残す。全言語を回したときだけ全面的に作り直す。
function saveBaseline(findings, ranLanguages) {
  const ran = new Set(ranLanguages)
  const isFullRun = Object.keys(LANGUAGES).every((l) => ran.has(l))
  let kept = []
  if (!isFullRun && isFile(BASELINE_FILE)) {
    const previous = JSON.parse(fs.readFileSync(BASELINE_FILE, 'utf8')).findings ?? []
    kept = previous.filter((f) => !ran.has(f.lang))
  }

  const body = {
    note: [
      'npm run codeql が「既知」として扱う指摘の一覧。ここに無い指摘が出たら失敗する。',
      'GitHub 側で棄却した指摘や、修正しないと決めた指摘を入れておく。',
      '更新するときは npm run codeql -- --update-baseline。',
      'lang は記録した言語。指定した言語ぶんだけが差し替わる。',
    ].join(' '),
    findings: [...kept, ...findings]
      .sort((a, b) => a.fingerprint.localeCompare(b.fingerprint))
      .map((f) => ({
        fingerprint: f.fingerprint,
        lang: f.lang,
        rule: f.rule,
        file: f.file,
        message: f.message,
      })),
  }
  fs.mkdirSync(path.dirname(BASELINE_FILE), { recursive: true })
  fs.writeFileSync(BASELINE_FILE, JSON.stringify(body, null, 2) + '\n')
}

function printFindings(title, findings) {
  console.log(`\n${title} ${findings.length}件:`)
  const byRule = new Map()
  for (const f of findings) byRule.set(f.rule, [...(byRule.get(f.rule) ?? []), f])
  for (const rule of [...byRule.keys()].sort()) {
    const group = byRule.get(rule)
    console.log(`  ${group.length}件  ${rule} [${group[0].severity}]`)
    for (const f of group) console.log(`      ${f.file}:${f.line}  ${f.message}`)
  }
}

// ── 本体 ───────────────────────────────────────────────────────────────

function main() {
  if (!isFile(CONFIG_FILE)) {
    console.error(`CodeQL の設定ファイルが見つからない: ${path.relative(ROOT, CONFIG_FILE)}`)
    return 1
  }
  // //go:embed embed は対象ディレクトリが空だとコンパイルエラーになる。
  // 追跡している PLACEHOLDER.md が満たしてくれるので locales の用意までは要らないが、
  // release の中断などで消えているとビルドの失敗として現れるので先に見ておく。
  const placeholder = path.join(ROOT, 'src', 'server', 'gkill', 'api', 'embed', 'PLACEHOLDER.md')
  if (languages.includes('go') && !isFile(placeholder)) {
    console.error('src/server/gkill/api/embed/PLACEHOLDER.md が無い。//go:embed が通らず Go のビルドが失敗する。')
    console.error('  git restore src/server/gkill/api/embed/PLACEHOLDER.md で戻すこと。')
    return 1
  }

  const cli = findCodeqlCli()
  if (!cli) {
    if (flags.has('--required')) {
      console.error('❌ CodeQL CLI が見つからない（--required 指定）')
      printInstallInstructions()
      return 1
    }
    console.log('')
    console.log('⏭  CodeQL CLI が見つからないのでスキップした。')
    printInstallInstructions()
    console.log('   CLI が無いことを失敗として扱いたいときは --required を付ける。')
    console.log('')
    return 0
  }

  // 成果物はリポジトリの外に置く。中に置くと .gitignore の管理対象が増えるうえ、
  // 数GBのデータベースがリポジトリ配下に居座ることになる。
  const workDir = process.env.GKILL_CODEQL_WORK || path.join(os.tmpdir(), 'gkill-codeql')
  fs.mkdirSync(workDir, { recursive: true })

  console.log(`CodeQL CLI : ${cli}`)
  console.log(`設定       : ${path.relative(ROOT, CONFIG_FILE)}（CI と共用）`)
  console.log(`作業場所   : ${workDir}`)
  console.log(`対象言語   : ${languages.join(', ')}`)

  const all = []
  for (const lang of languages) {
    const dbDir = path.join(workDir, `db-${lang}`)
    const sarifPath = path.join(workDir, `${lang}.sarif`)

    // Kotlin の固定はビルドするときだけ行う。データベースを再利用するときに
    // 巻き込むと、追跡ファイルを無駄に書き換えるうえ、libs.versions.toml に
    // 未コミットの変更があるだけで「クエリを回し直すだけ」まで止まってしまう。
    let created
    if (flags.has('--reuse-db') && fs.existsSync(dbDir)) {
      console.log(`\n=== データベースを再利用 (${lang}): ${dbDir}`)
      created = 0
    } else if (LANGUAGES[lang].buildMode === 'manual') {
      created = withKotlinPin(() => createDatabase(cli, lang, dbDir))
    } else {
      created = createDatabase(cli, lang, dbDir)
    }
    if (created !== 0) {
      console.error(`❌ データベースの作成に失敗した: ${lang}`)
      return 1
    }
    if (analyzeDatabase(cli, lang, dbDir, sarifPath) !== 0) {
      console.error(`❌ 解析に失敗した: ${lang}`)
      return 1
    }
    all.push(...readSarif(sarifPath, lang))
    console.log(`   SARIF: ${sarifPath}`)
  }

  if (flags.has('--update-baseline')) {
    saveBaseline(all, languages)
    printFindings('📝 ベースラインに記録した', all)
    console.log(`\n→ ${path.relative(ROOT, BASELINE_FILE)}`)
    return 0
  }

  const baselineEntries = loadBaseline()
  if (!baselineEntries) {
    printFindings('📋 検出', all)
    console.log('')
    console.log(`ℹ ベースライン（${path.relative(ROOT, BASELINE_FILE)}）が無いので合否は判定しない。`)
    console.log('  いまの結果を既知として記録するなら: npm run codeql -- --update-baseline')
    return 0
  }

  const known = new Set(baselineEntries.map((f) => f.fingerprint))
  const fresh = all.filter((f) => !known.has(f.fingerprint))
  const seen = new Set(all.map((f) => f.fingerprint))
  // 「今回出なかった」は走らせた言語ぶんだけ数える。
  // 走らせていない言語の記録まで数えると、単一言語で回すたびに必ず出て意味を失う。
  const ran = new Set(languages)
  const gone = baselineEntries.filter((f) => ran.has(f.lang) && !seen.has(f.fingerprint))

  if (gone.length > 0) {
    console.log(`\n✅ ベースラインにあって今回出なかった指摘: ${gone.length}件（直ったなら --update-baseline で減らす）`)
  }
  if (fresh.length > 0) {
    printFindings('❌ ベースラインに無い指摘', fresh)
    console.log('')
    console.log('  意図した指摘なら --update-baseline で既知に加える。')
    return 1
  }
  console.log(`\n✅ 新しい指摘なし（既知 ${all.length}件のみ）`)
  return 0
}

process.exit(main())
