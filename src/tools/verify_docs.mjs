#!/usr/bin/env node
// ドキュメント検証ツール（docs CI）
//
// 目的: 手書きの件数・相互リンク・参照パス・Mermaid をコードから検証し、
//       ドリフト（コード変更で docs の記載が陳腐化する事故）を防ぐ。
//
// 使い方:
//   node src/tools/verify_docs.mjs         検証（不整合があれば exit 1）
//   node src/tools/verify_docs.mjs --list  実測メトリクスを表示して終了
//
// 依存なし（Node 標準のみ）。リポジトリルートからでも任意の CWD からでも動作する。
//
// ファイル名の実在まで検査する理由と、除外を2種類に絞った理由:
// documents/adr/0803-verify-docs-checks-filenames.md

import { execSync } from 'node:child_process'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import zlib from 'node:zlib'
import { renderAll, OUT_DIR, SRC_DIR } from './manual_build.mjs'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..', '..') // src/tools/ → リポジトリルート

// ─────────────────────────────────────────────────────────────
// ユーティリティ
// ─────────────────────────────────────────────────────────────
const abs = (p) => path.join(ROOT, p)
const readText = (p) => fs.readFileSync(abs(p), 'utf8')
const exists = (p) => fs.existsSync(abs(p))

function listFiles(dir, filter) {
  const full = abs(dir)
  if (!fs.existsSync(full)) return []
  return fs.readdirSync(full).filter(filter)
}

// dir 直下の該当ファイル数。suffix で絞り、excludeTest でテストを除く。
function countIn(dir, suffix, { excludeTest = false, prefix = '' } = {}) {
  return listFiles(dir, (f) =>
    f.endsWith(suffix) &&
    f.startsWith(prefix) &&
    !(excludeTest && (f.endsWith('_test' + suffix) || f.endsWith('.test' + suffix) || f.endsWith('.spec' + suffix)))
  ).length
}

// dir 以下を再帰的に走査してフルパス配列を返す。
function listFilesRec(dir, filter) {
  const full = abs(dir)
  if (!fs.existsSync(full)) return []
  const out = []
  const walk = (d) => {
    for (const e of fs.readdirSync(d, { withFileTypes: true })) {
      const p = path.join(d, e.name)
      if (e.isDirectory()) walk(p)
      else if (filter(e.name, p)) out.push(p)
    }
  }
  walk(full)
  return out
}

// ファイルの行数。docs が「約N行」と書いている値の実測に使う。
function lineCount(rel) {
  if (!exists(rel)) return 0
  return readText(rel).split(/\r?\n/).length
}

// ファイル群に正規表現が何回マッチするかの合計（静的計数）。
function countMatches(files, re) {
  let n = 0
  for (const f of files) {
    const m = fs.readFileSync(f, 'utf8').match(re)
    if (m) n += m.length
  }
  return n
}

const errors = []
const warnings = []
const err = (m) => errors.push(m)
const warn = (m) => warnings.push(m)

// ─────────────────────────────────────────────────────────────
// 1. 実測メトリクスの計算
// ─────────────────────────────────────────────────────────────
function computeMetrics() {
  const handlers = listFiles('src/server/gkill/api/gkill_server_api',
    (f) => f.startsWith('handle_') && f.endsWith('.go')).length
  const reqRes = listFiles('src/server/gkill/api/req_res',
    (f) => f.endsWith('.go')).length
  const views = listFiles('src/client/pages/views', (f) => f.endsWith('.vue')).length
  const dialogs = listFiles('src/client/pages/dialogs', (f) => f.endsWith('.vue')).length
  const pages = listFiles('src/client/pages', (f) => f.endsWith('.vue')).length

  // エンドポイント数 = gkill_server_api_address.go のルート表（apiRoutes）の行数。
  //   表は serve.go とテストハーネスがそのまま登録する正本なので「定義数」と「登録数」は同じ数。
  //   かつては定義（アドレス定数）と登録（serve.go の HandleFunc）が別で、「92件定義・90件登録」
  //   という二重の数を資料に書かせていた（ADR-0709）。
  //   行の書式 `{Path: "...", Method: "...", Auth: ..., Body: ..., Handler: g.HandleXxx},` は
  //   表の冒頭コメントで契約している（TS 側の gkill-api.test.ts も同じ正規表現で読む）。
  const addressGo = readText('src/server/gkill/api/gkill_server_api/gkill_server_api_address.go')
  const routeRows = [...addressGo.matchAll(/\{Path:\s*"([^"]+)",\s*Method:\s*"(GET|POST)",\s*Auth:\s*(\w+),/g)]
  const endpoints = routeRows.length
  const endpointsPost = routeRows.filter((m) => m[2] === 'POST').length
  const endpointsGet = routeRows.filter((m) => m[2] === 'GET').length
  if (endpoints === 0) err('ルート表（gkill_server_api_address.go の apiRoutes）を読み取れない。行の書式が変わったなら verify_docs の正規表現も直すこと')

  // 認証区分別の件数。表の authNone / authSession / authSessionRepos は
  // auth_middleware.go の wrapNoAuth（capped 含む）/ wrapAuth / wrapAuthRepos と1対1。
  const authCount = (kind) => routeRows.filter((m) => m[3] === kind).length
  const wrapNoAuth = authCount('authNone')
  const wrapAuth = authCount('authSession')
  const wrapAuthRepos = authCount('authSessionRepos')

  // i18n キー数（全ロケール一致を検査、ja を代表値に）
  const localeFiles = listFiles('src/locales', (f) => f.endsWith('.json'))
  const localeKeyCounts = {}
  for (const f of localeFiles) {
    try {
      const obj = JSON.parse(readText(path.join('src/locales', f)))
      localeKeyCounts[f] = Object.keys(obj).length
    } catch (e) {
      err(`locale JSON parse 失敗: src/locales/${f}: ${e.message}`)
    }
  }
  const i18nKeys = localeKeyCounts['ja.json']

  return {
    handlers, reqRes, views, dialogs, pages, endpoints, i18nKeys,
    endpointsPost, endpointsGet,
    wrapNoAuth, wrapAuth, wrapAuthRepos,
    componentTotal: views + dialogs + pages,
    localeKeyCounts,
    ...computeDirMetrics(),
    ...computeTestMetrics(),
    ...computeMiscMetrics(),
  }
}

// ─────────────────────────────────────────────────────────────
// 1-b. ディレクトリ別ファイル数
//   README.md 群が「（N ファイル）」と書いている数値の実測。
//   テストを含むか否かは doc 側の書き方に合わせて両方を出す。
// ─────────────────────────────────────────────────────────────
function computeDirMetrics() {
  const SA = 'src/server/gkill/api/gkill_server_api'
  const DN = 'src/client/classes/dnote'
  return {
    // サーバ
    serverApiGo: countIn(SA, '.go'),
    serverApiTest: listFiles(SA, (f) => f.endsWith('_test.go')).length,
    // handlers（既存）はテストも含む handle_*.go。実装本数は別に出す。
    handlersImpl: listFiles(SA, (f) => f.startsWith('handle_') && f.endsWith('.go') && !f.endsWith('_test.go')).length,
    serverApiBase: listFiles(SA, (f) => f.endsWith('.go') && !f.startsWith('handle_') && !f.endsWith('_test.go')).length,
    reqResType: countIn('src/server/gkill/api/req_res', '.go', { excludeTest: true }),
    kftlGo: countIn('src/server/gkill/api/kftl', '.go'),
    // message/README.md の「ファイル一覧」はテストも含めて数えている。
    messageGo: countIn('src/server/gkill/api/message', '.go'),
    repsGo: countIn('src/server/gkill/dao/reps', '.go'),
    repsImpl: countIn('src/server/gkill/dao/reps', '.go', { excludeTest: true }),
    // usecase/README.md が数えているのは本番の実装面。
    // テストファイルを混ぜると「ファイル一覧」に載っていないファイルまで数に入り、
    // 「Add 系 12 + Update 系 13 + ...」という関数の内訳とも合わなくなる
    usecaseGo: countIn('src/server/gkill/usecase', '.go', { excludeTest: true }),
    usecaseFuncs: countMatches(
      listFilesRec('src/server/gkill/usecase', (f) => f.endsWith('.go') && !f.endsWith('_test.go')), /^func /gm),
    sdkGo: countIn('src/server/gkill/plugin/sdk', '.go'),
    // クライアント
    classesRoot: countIn('src/client/classes', '.ts'),
    classesUse: countIn('src/client/classes', '.ts', { prefix: 'use-' }),
    // gkill-api.ts は編集のたびに数行動くので、100の位に丸めた「約N,N00行」を検査する。
    // 素の行数で検査すると1行足すたびに docs CI が落ちて、かえって形骸化する。
    gkillApiLinesApprox: Math.round(lineCount('src/client/classes/api/gkill-api.ts') / 100) * 100,
    classesApiRoot: countIn('src/client/classes/api', '.ts'),
    classesApiReqRes: countIn('src/client/classes/api/req_res', '.ts'),
    classesDatasRoot: countIn('src/client/classes/datas', '.ts'),
    kftlTsRoot: countIn('src/client/classes/kftl', '.ts'),
    dnoteRoot: countIn(DN, '.ts'),
    dnoteAggregateTarget: countIn(`${DN}/dnote-aggregate-target`, '.ts'),
    dnotePredicate: countIn(`${DN}/dnote-predicate`, '.ts'),
    dnoteTargetKyouPredicate: countIn(`${DN}/dnote-predicate/target-kyou-predicate`, '.ts'),
    dnoteKeyGetter: countIn(`${DN}/dnote-key-getter`, '.ts'),
    // ビュー / ダイアログの接頭辞別
    viewsAdd: countIn('src/client/pages/views', '.vue', { prefix: 'add-' }),
    viewsEdit: countIn('src/client/pages/views', '.vue', { prefix: 'edit-' }),
    viewsConfirm: countIn('src/client/pages/views', '.vue', { prefix: 'confirm-' }),
    dialogsAdd: countIn('src/client/pages/dialogs', '.vue', { prefix: 'add-' }),
    dialogsEdit: countIn('src/client/pages/dialogs', '.vue', { prefix: 'edit-' }),
    dialogsConfirm: countIn('src/client/pages/dialogs', '.vue', { prefix: 'confirm-' }),
  }
}

// ─────────────────────────────────────────────────────────────
// 1-c. テスト件数（すべて「静的計数」）
//   実行件数は .each 展開 / skip / プロジェクト数で変わるため一致しない。
//   doc 側にも「静的計数」と明記して、実行結果と混同させないこと。
// ─────────────────────────────────────────────────────────────
const GO_TEST_RE = /^func Test/gm
const VITEST_RE = /^\s*(it|test)(\.each)?\(/gm
const PW_TEST_RE = /^\s*test\(/gm

function computeTestMetrics() {
  const goTestFiles = listFilesRec('src/server', (f) => f.endsWith('_test.go'))
  const goPkgs = new Set(goTestFiles.map((f) => path.dirname(f)))
  const unitFiles = listFilesRec('src/client/__tests__/unit', (f) => f.endsWith('.test.ts'))
  const e2eFiles = listFilesRec('src/client/__tests__/e2e', (f) => f.endsWith('.spec.ts'))
  const mcpFiles = listFilesRec('src/mcp/__tests__', (f) => /\.test\.(mjs|js|ts)$/.test(f))
  const kt = (dir) => countMatches(listFilesRec(dir, (f) => f.endsWith('.kt')), /@Test/g)
  // @Test を1つでも含む .kt の本数（＝テストファイル数）
  const ktFiles = (dir) => listFilesRec(dir, (f) => f.endsWith('.kt'))
    .filter((f) => /@Test/.test(fs.readFileSync(f, 'utf8'))).length

  // ABOUT_TEST.md 群がディレクトリ単位で「（Nファイル）」と書いている数の実測。
  const unitDirFiles = (sub) =>
    listFiles(`src/client/__tests__/unit/${sub}`, (f) => f.endsWith('.test.ts')).length

  const m = {
    goTests: countMatches(goTestFiles, GO_TEST_RE),
    goTestFiles: goTestFiles.length,
    goTestPkgs: goPkgs.size,
    unitTests: countMatches(unitFiles, VITEST_RE),
    unitTestFiles: unitFiles.length,
    e2eTests: countMatches(e2eFiles, PW_TEST_RE),
    e2eTestFiles: e2eFiles.length,
    mcpTests: countMatches(mcpFiles, VITEST_RE),
    mcpTestFiles: mcpFiles.length,
    pluginGoTests: countMatches(
      listFilesRec('src/plugins', (f) => f.endsWith('_test.go')), GO_TEST_RE),
    sdkTests: countMatches(
      listFilesRec('src/server/gkill/plugin/sdk', (f) => f.endsWith('_test.go')), GO_TEST_RE),
    sdkTestFiles: listFilesRec(
      'src/server/gkill/plugin/sdk', (f) => f.endsWith('_test.go')).length,
    wearCompanionTests: kt('src/wear_os/phone_companion'),
    wearWatchTests: kt('src/wear_os/watch_app'),
    androidTests: kt('src/android'),
    androidTestFiles: ktFiles('src/android'),
    wearTestFiles: ktFiles('src/wear_os/phone_companion') + ktFiles('src/wear_os/watch_app'),
    unitClassesFiles: unitDirFiles('classes'),
    unitComposablesFiles: unitDirFiles('composables'),
    unitApiFiles: unitDirFiles('api'),
    unitDatasFiles: unitDirFiles('datas'),
    unitDnoteFiles: unitDirFiles('dnote'),
    unitKftlFiles: unitDirFiles('kftl'),
    serverApiTestFiles: listFiles('src/server/gkill/api/gkill_server_api',
      (f) => f.endsWith('_test.go')).length,
    serverMainTestFiles: listFilesRec('src/server/gkill/main', (f) => f.endsWith('_test.go')).length,
  }

  // ABOUT_TEST.md の「合計」行。手計算で合わないまま放置されやすいので実測から出す。
  // Wear OS は phone_companion + watch_app の2モジュール分。
  m.totalTests = m.goTests + m.unitTests + m.e2eTests + m.mcpTests +
    m.androidTests + m.wearCompanionTests + m.wearWatchTests
  m.totalTestFiles = m.goTestFiles + m.unitTestFiles + m.e2eTestFiles + m.mcpTestFiles +
    m.androidTestFiles + m.wearTestFiles
  return m
}

// ─────────────────────────────────────────────────────────────
// 1-d. その他（MCPツール数 / KFTLステートメント型数 / 用語集件数）
// ─────────────────────────────────────────────────────────────
function computeMiscMetrics() {
  // MCPサーバの TOOLS は lib/*-tools.mjs のスプレッドで組み立てるので、
  // サーバ本体のファイルだけを見ても数えられない。スプレッドを辿って数える。
  // 書き込みサーバは読み取りツールの一部だけを載せるので、絞り込みの名前集合も見る。
  const TOOL_MODULES = {
    READ_TOOLS: 'src/mcp/lib/read-tools.mjs',
    WRITE_TOOLS: 'src/mcp/lib/write-tools.mjs',
    PLUGIN_TOOLS: 'src/mcp/lib/plugin-tools.mjs',
  }
  const namesIn = (rel) => (exists(rel)
    ? [...readText(rel).matchAll(/name: *"(gkill_[a-z_0-9]+)"/g)].map((m) => m[1])
    : [])
  const toolNames = (rel) => {
    if (!exists(rel)) return 0
    const src = readText(rel)
    const names = new Set(namesIn(rel))
    const toolsBlock = src.match(/const TOOLS = \[[\s\S]*?\n\];/)
    if (!toolsBlock) return names.size
    for (const spread of toolsBlock[0].matchAll(/\.\.\.(\w+)(?:\.filter\([^\n]*?(\w+)\.has)?/g)) {
      const mod = TOOL_MODULES[spread[1]]
      if (!mod) continue
      let modNames = namesIn(mod)
      if (spread[2]) {
        const setBlock = src.match(new RegExp(`const ${spread[2]} = new Set\\(\\[([\\s\\S]*?)\\]\\)`))
        if (setBlock) {
          const allow = new Set([...setBlock[1].matchAll(/"(gkill_[a-z_0-9]+)"/g)].map((x) => x[1]))
          modNames = modNames.filter((n) => allow.has(n))
        }
      }
      for (const n of modNames) names.add(n)
    }
    return names.size
  }
  // ステートメント型 = 名前が StatementLine で終わる型のうち、基底の KFTLStatementLine を除いたもの。
  const BASE = 'KFTLStatementLine'
  const kftlTs = new Set(listFilesRec('src/client/classes/kftl', (f) => f.endsWith('.ts'))
    .flatMap((f) => [...fs.readFileSync(f, 'utf8')
      .matchAll(/export (?:abstract )?class (\w*StatementLine)\b/g)].map((m) => m[1])))
  const kftlGo = new Set(listFilesRec('src/server/gkill/api/kftl', (f) => f.endsWith('.go'))
    .flatMap((f) => [...fs.readFileSync(f, 'utf8')
      .matchAll(/^type (\w*StatementLine) (?:struct|interface)\b/gm)].map((m) => m[1])))
  kftlTs.delete(BASE)
  kftlGo.delete(BASE)

  // glossary の用語数 = §1-9 の表で先頭セルが太字の行（§10 は相互参照表なので除外）
  let glossaryTerms = 0
  if (exists('documents/reverse/glossary.md')) {
    const body = readText('documents/reverse/glossary.md').split('## 10.')[0]
    glossaryTerms = (body.match(/^\| \*\*[^*|]+\*\*/gm) || []).length
  }

  // usecase.md のユースケース数 = ユニークな UC-ID 数（延べ出現数ではない）
  const ucIds = exists('documents/reverse/usecase.md')
    ? new Set(readText('documents/reverse/usecase.md').match(/UC-\d{4}/g) || []).size
    : 0

  // sequence-diagrams.md のシーケンス図数 = mermaid ブロック数
  const seqDiagrams = exists('documents/reverse/sequence-diagrams.md')
    ? (readText('documents/reverse/sequence-diagrams.md').match(/```mermaid/g) || []).length
    : 0

  // エラー/メッセージコードのユニーク定数数
  const codeCount = (rel, re) => (exists(rel) ? new Set(readText(rel).match(re) || []).size : 0)
  const errCodes = codeCount('src/server/gkill/api/message/error_codes.go', /ERR000\d+/g)
  const msgCodes = codeCount('src/server/gkill/api/message/message_codes.go', /MSG000\d+/g)

  // 採番の上端。件数だけ検査していると「406定数、ERR000001〜ERR000401」のように
  // 件数は合っているのに範囲が古い、という書き方が通り抜けるので別途検査する。
  const codeMax = (rel, re) => {
    if (!exists(rel)) return ''
    const all = [...new Set(readText(rel).match(re) || [])].sort()
    return all.length ? all[all.length - 1] : ''
  }
  const errCodeMax = codeMax('src/server/gkill/api/message/error_codes.go', /ERR000\d+/g)
  const msgCodeMax = codeMax('src/server/gkill/api/message/message_codes.go', /MSG000\d+/g)

  // クライアント専用のエラーコード（ERR9xxxxx）。Go 側の error_codes.go には存在しない帯。
  const clientErrCodes = codeCount('src/client/classes/api/message/gkill_error.ts', /ERR9\d{5}/g)
  const clientErrCodeMax = codeMax('src/client/classes/api/message/gkill_error.ts', /ERR9\d{5}/g)

  // FindQuery のフィールド数（json タグ付き。json:"-" の ExcludeURLogThumbnailImage も含む）
  let findQueryFields = 0
  if (exists('src/server/gkill/api/find/find_query.go')) {
    const structBody = readText('src/server/gkill/api/find/find_query.go')
      .match(/type FindQuery struct \{[\s\S]*?\n\}/)
    findQueryFields = structBody ? (structBody[0].match(/json:"/g) || []).length : 0
  }

  // go.mod の Go バージョン
  const goVersion = exists('src/server/go.mod')
    ? (readText('src/server/go.mod').match(/^go (\d+\.\d+(?:\.\d+)?)/m) || [])[1] || ''
    : ''

  // docコメント網羅率。
  //   reps: *_repository.go のインターフェースメソッド宣言（タブ + 大文字始まり + 「(」）
  //   handlers: func (g *GkillServerAPI) HandleXxx
  // どちらも「直前行が // で始まるか」で判定する。
  const docCoverage = (files, declRe) => {
    let total = 0
    let documented = 0
    for (const file of files) {
      const lines = readText(file).split(/\r?\n/)
      for (let i = 0; i < lines.length; i++) {
        if (!declRe.test(lines[i])) continue
        total++
        if (i > 0 && /^\s*\/\//.test(lines[i - 1])) documented++
      }
    }
    return { total, documented }
  }
  const repsIfaceFiles = listFiles('src/server/gkill/dao/reps', (f) => f.endsWith('_repository.go'))
    .map((f) => path.join('src/server/gkill/dao/reps', f))
  const repsIfaceDocs = docCoverage(repsIfaceFiles, /^\t[A-Z]\w*\(/)
  const handlerFiles = listFiles('src/server/gkill/api/gkill_server_api', (f) => f.endsWith('.go'))
    .map((f) => path.join('src/server/gkill/api/gkill_server_api', f))
  const handlerDocs = docCoverage(handlerFiles, /^func \(g \*GkillServerAPI\) Handle/)

  // 書き込み後のキャッシュ反映（WriteThroughXxxCache）の呼び出し件数。
  //   CLAUDE.md が「N箇所」と書いているのはこの数。定義側（GkillRepositories のメソッド）と
  //   テストは除く。反映を飛ばすと最大1分だけ古い応答が見え、PWA が焼き付ける事故につながるので、
  //   件数が動いたら CLAUDE.md も見直す、を機械で促す。
  const writeThroughCalls = listFilesRec('src/server/gkill', (f) => f.endsWith('.go') && !f.endsWith('_test.go'))
    .reduce((sum, f) => {
      // 定義側（`func (r *GkillRepositories) WriteThroughXxxCache(`）とインターフェース宣言には
      // 先行する `.` が無いので、この正規表現には最初から掛からない。
      const src = fs.readFileSync(f, 'utf8')
      return sum + (src.match(/\.WriteThrough\w*Cache\(/g) || []).length
    }, 0)

  // dao/reps 直下のテストファイル数 / クライアント datas テストファイル数 / Wear OS Kotlin ファイル数
  const repsTestFiles = listFiles('src/server/gkill/dao/reps', (f) => f.endsWith('_test.go')).length
  // サブパッケージ（reps/cache, reps/rep_cache_updater）を含めた本数。
  // ABOUT_TEST.md の「リポジトリ」行はこちらの数を書いている
  const repsAllTestFiles = listFilesRec('src/server/gkill/dao/reps', (f) => f.endsWith('_test.go')).length
  const datasTestFiles = listFiles('src/client/__tests__/unit/datas', (f) => f.endsWith('.test.ts')).length
  const wearCompanionKt = listFilesRec('src/wear_os/phone_companion/src/main', (f) => f.endsWith('.kt')).length
  const wearWatchKt = listFilesRec('src/wear_os/watch_app/src/main', (f) => f.endsWith('.kt')).length

  // マニュアルのページ数（1言語あたり）。ja を代表値にする。
  // 言語間の集合一致は checkManuals() が別に見ているので、ここは枚数だけ。
  const manualPages = exists('resources/manual_src/ja')
    ? listFiles('resources/manual_src/ja', (f) => f.endsWith('.html')).length
    : 0

  // ルータのルート数。
  //   コンポーネントを持つルートと、旧パス吸収の redirect 専用ルートは別物。
  //   `/shared_mi` がコンポーネントから redirect に変わったとき、
  //   「13ルート」「14ルート」「表14行」が資料の中で三重に食い違った。
  const routerSource = exists('src/client/router/index.ts') ? readText('src/client/router/index.ts') : ''
  const routeEntries = routerSource.split(/\n\s*\{\s*\n/).slice(1)
  const routeComponents = routeEntries.filter((e) => /^\s*component:/m.test(e)).length
  const routeRedirects = routeEntries.filter((e) =>
    /^\s*redirect:/m.test(e) && !/^\s*component:/m.test(e)).length

  // go.mod を持つモジュール数（src/server + src/plugins の各プラグイン）。
  // 依存を上げるときは全モジュールで tidy が要るので、資料の数が古いと取りこぼす。
  const goModModules = listFilesRec('src', (f) => f === 'go.mod').length

  return {
    writeThroughCalls,
    manualPages,
    routeComponents,
    routeRedirects,
    goModModules,
    mcpReadTools: toolNames('src/mcp/gkill-read-server.mjs'),
    mcpWriteTools: toolNames('src/mcp/gkill-write-server.mjs'),
    mcpReadWriteTools: toolNames('src/mcp/gkill-readwrite-server.mjs'),
    mcpPluginTools: toolNames('src/mcp/lib/plugin-tools.mjs'),
    kftlStatementTs: kftlTs.size,
    kftlStatementGo: kftlGo.size,
    glossaryTerms,
    ucIds,
    seqDiagrams,
    errCodes,
    msgCodes,
    errCodeMax,
    msgCodeMax,
    clientErrCodes,
    clientErrCodeMax,
    findQueryFields,
    goVersion,
    repsIfaceMethods: repsIfaceDocs.total,
    repsIfaceDocumented: repsIfaceDocs.documented,
    handlerMethods: handlerDocs.total,
    handlerDocumented: handlerDocs.documented,
    repsTestFiles,
    repsAllTestFiles,
    datasTestFiles,
    wearCompanionKt,
    wearWatchKt,
    adrCount: listFiles('documents/adr',
      (f) => /^\d{4}-.+\.md$/.test(f) && f !== '0000-template.md').length,
    skillCount: listFiles('.claude/skills',
      (d) => exists('.claude/skills/' + d + '/SKILL.md')).length,
    agentsMdBytes: exists('AGENTS.md')
      ? Buffer.byteLength(readText('AGENTS.md').replace(/\r\n/g, '\n'), 'utf8') : 0,
  }
}

// ─────────────────────────────────────────────────────────────
// 2. 件数の突合（各 doc に「正しい件数を含む語句」が存在するか）
//    ドリフト時: コードが変わると期待語句が変化し、doc に無いので FAIL。
// ─────────────────────────────────────────────────────────────
function buildCountAssertions(m) {
  const A = []
  const add = (file, phrase) => A.push({ file, phrase })

  // handlers
  add('documents/reverse/api-endpoints.md', `1ハンドラ1ファイル、${m.handlers}ファイル`)
  add('documents/reverse/folder-structure.md', `1ファイル1ハンドラ、${m.handlers}ファイル`)
  add('documents/reverse/glossary.md', `handle_*.go ${m.handlers}ファイル`)
  add('documents/reverse/program-spec.md', `handle_*.go ${m.handlers}ファイル`)

  // req_res
  add('documents/reverse/folder-structure.md', `構造体（${m.reqRes}ファイル）`)
  add('documents/reverse/glossary.md', `入出力構造体（${m.reqRes}ファイル）`)
  add('documents/reverse/frontend-architecture.md', `サーバー側は${m.reqRes}ファイル`)

  // views
  add('documents/reverse/folder-structure.md', `ビューコンポーネント（${m.views}ファイル）`)
  add('documents/reverse/screen-specs.md', `| ビューコンポーネント | ${m.views} |`)
  add('documents/reverse/program-spec.md', `| ビュー | ${m.views} |`)

  // dialogs
  add('documents/reverse/screen-specs.md', `| ダイアログコンポーネント | ${m.dialogs} |`)
  add('documents/reverse/program-spec.md', `| ダイアログ | ${m.dialogs} |`)
  add('documents/reverse/frontend-architecture.md', `${m.dialogs}ダイアログ中`)
  add('.claude/skills/gkill-client-foundation/SKILL.md', `${m.dialogs} dialog components`)

  // i18n keys
  add('documents/reverse/folder-structure.md', `${m.i18nKeys}キー/言語`)
  add('AGENTS.md', `${m.i18nKeys} keys per locale`)

  // endpoints（ルート表の行数）
  add('documents/reverse/glossary.md', `全エンドポイント（${m.endpoints}件）`)
  add('documents/reverse/program-spec.md', `全${m.endpoints}エンドポイント`)

  // コンポーネント合計
  add('documents/reverse/screen-specs.md', `**${m.componentTotal}**（ルートを除く`)

  // ── src/**/README.md（ディレクトリ別ファイル数）
  add('src/README.md', `${m.endpoints} POST API エンドポイント`)
  add('src/README.md', `HTTP API (${m.endpoints}エンドポイント)`)
  add('src/README.md', `View コンポーネント (${m.views}個)`)
  add('src/README.md', `Dialog コンポーネント (${m.dialogs}個)`)
  add('src/README.md', `Request/Response 構造体 (${m.reqRes}ファイル)`)
  add('src/server/README.md', `全${m.endpoints}エンドポイント一覧`)
  add('src/server/README.md', `HTTP ハンドラ（${m.serverApiGo}ファイル）`)
  add('src/server/gkill/api/README.md', `HTTP ハンドラ（${m.serverApiGo}ファイル）`)
  add('src/server/gkill/api/README.md', `全${m.endpoints}エンドポイントを登録`)
  add('src/server/gkill/api/README.md', `### \`kftl/\`（${m.kftlGo}ファイル）`)
  add('src/server/gkill/api/README.md', `### \`req_res/\`（${m.reqRes}ファイル）`)
  add('src/server/gkill/api/message/README.md', `## ファイル一覧（${m.messageGo}ファイル）`)
  add('src/server/gkill/api/gkill_server_api/README.md', `ハンドラ実装${m.handlersImpl}`)
  add('src/server/gkill/api/req_res/README.md', `（${m.reqRes}ファイル: 型定義${m.reqResType} +`)
  add('src/server/gkill/dao/README.md', `### \`reps/\`（${m.repsImpl}ファイル。テストを含めると${m.repsGo}）`)
  add('src/server/gkill/usecase/README.md', `## ファイル一覧（${m.usecaseGo}ファイル）`)
  add('src/server/gkill/usecase/README.md', `**合計: ${m.usecaseFuncs} 関数**`)
  add('src/server/gkill/plugin/README.md', `Go SDK（${m.sdkGo}ファイル`)
  add('src/client/classes/README.md', `(ルートファイル ${m.classesRoot}個)`)
  add('src/client/classes/api/README.md', `### \`req_res/\`（${m.classesApiReqRes}ファイル）`)
  add('src/client/classes/datas/README.md', `## ルートファイル（${m.classesDatasRoot}ファイル）`)
  add('src/client/classes/kftl/README.md', `## ルートファイル（${m.kftlTsRoot}ファイル）`)
  add('src/client/classes/dnote/README.md', `\`dnote-aggregate-target/\`（${m.dnoteAggregateTarget}ファイル）`)
  add('src/client/classes/dnote/README.md', `\`dnote-predicate/\`（${m.dnotePredicate}ファイル）`)
  add('src/client/pages/README.md', `View コンポーネント一覧（${m.views} .vue）`)
  add('src/client/pages/README.md', `Dialog コンポーネント一覧（${m.dialogs} .vue）`)
  add('src/client/pages/README.md', `### データ追加系（${m.viewsAdd}コンポーネント）`)
  add('src/client/pages/README.md', `### データ編集系（${m.viewsEdit}コンポーネント）`)
  add('src/client/pages/README.md', `### 削除確認系（${m.viewsConfirm}コンポーネント）`)
  add('src/client/pages/README.md', `### データ追加ダイアログ（${m.dialogsAdd}コンポーネント）`)
  add('src/client/pages/README.md', `### データ編集ダイアログ（${m.dialogsEdit}コンポーネント）`)
  add('src/client/pages/README.md', `### 削除確認ダイアログ（${m.dialogsConfirm}コンポーネント）`)

  // ── テスト件数（静的計数）。実行件数ではないので doc 側にもその旨を書くこと。
  const testRows = [
    `| Go バックエンド (\`server/\`) | ${m.goTests} | ${m.goTestFiles} |`,
    `| フロントエンド ユニット (\`client/\`) | ${m.unitTests} | ${m.unitTestFiles} |`,
    `| フロントエンド E2E (\`client/\`) | ${m.e2eTests} | ${m.e2eTestFiles} |`,
    `| MCP サーバ (\`mcp/\`) | ${m.mcpTests} | ${m.mcpTestFiles} |`,
    `| Android (\`android/\`) | ${m.androidTests} | ${m.androidTestFiles} |`,
    `| Wear OS (\`wear_os/\`) | ${m.wearCompanionTests + m.wearWatchTests} | ${m.wearTestFiles} |`,
  ]
  for (const row of testRows) add('src/ABOUT_TEST.md', row)
  add('src/ABOUT_TEST.md', `Go テスト ${m.pluginGoTests}件`)
  add('documents/reverse/testing-guide.md', `| Go バックエンド | ${m.goTests} | ${m.goTestFiles} |`)
  add('documents/reverse/testing-guide.md', `| フロントエンド ユニット | ${m.unitTests} | ${m.unitTestFiles} |`)
  add('documents/reverse/testing-guide.md', `| フロントエンド E2E | ${m.e2eTests} |`)
  add('documents/reverse/testing-guide.md', `| MCP サーバ | ${m.mcpTests} | ${m.mcpTestFiles} |`)
  add('documents/reverse/testing-guide.md', `| Android | ${m.androidTests} | ${m.androidTestFiles} |`)
  add('documents/reverse/testing-guide.md', `| Wear OS | ${m.wearCompanionTests + m.wearWatchTests} | ${m.wearTestFiles} |`)
  add('src/server/ABOUT_TEST.md', `${m.goTests}テスト関数、${m.goTestFiles}テストファイル、${m.goTestPkgs}パッケージ`)
  add('src/server/gkill/plugin/sdk/ABOUT_TEST.md', `**${m.sdkTests}テスト（${m.sdkTestFiles}ファイル）**`)
  add('src/mcp/ABOUT_TEST.md', `${m.mcpTests}テスト（${m.mcpTestFiles}ファイル）`)
  add('src/wear_os/ABOUT_TEST.md', `合計${m.wearCompanionTests + m.wearWatchTests}テスト`)

  // ── MCP ツール数。toolNames がスプレッド (lib/*-tools.mjs) を辿るので、
  //    プラグインツールもそこに含まれる（別途足さない）
  const mcpRead = m.mcpReadTools
  const mcpWrite = m.mcpWriteTools
  const mcpRW = m.mcpReadWriteTools
  add('.claude/skills/gkill-mcp/SKILL.md', `| Read | ${mcpRead} (`)
  add('.claude/skills/gkill-mcp/SKILL.md', `| Write | ${mcpWrite} (`)
  add('.claude/skills/gkill-mcp/SKILL.md', `| ReadWrite | ${mcpRW} (`)
  add('resources/manual_src/ja/mcp.html', `<td>gkill-read-server.mjs</td><td>${mcpRead}</td>`)
  add('resources/manual_src/ja/mcp.html', `<td>gkill-write-server.mjs</td><td>${mcpWrite}</td>`)
  add('resources/manual_src/ja/mcp.html', `<td>gkill-readwrite-server.mjs</td><td>${mcpRW}</td>`)

  // README / ABOUT_TEST は本数を「プラグイン1本を除いた内訳」でも書いている。
  // そこが検査から漏れていたため 11/26/32・10/24/30・Read 9/Write 24/ReadWrite 29 と
  // 3世代ぶんのドリフトが同時に残っていた（2026-08-24 の再監査で発見）。内訳まで検査する。
  const mcpReadOnly = mcpRead - 1
  const mcpWriteOnly = mcpRW - mcpRead
  const mcpWriteConvenience = mcpWrite - mcpWriteOnly - 1
  add('src/mcp/README.md', `\`gkill-read-server.mjs\` | ${mcpRead} (${mcpReadOnly} read + 1 plugin)`)
  add('src/mcp/README.md', `\`gkill-write-server.mjs\` | ${mcpWrite} (${mcpWriteOnly} write + ${mcpWriteConvenience} read convenience + 1 plugin)`)
  add('src/mcp/README.md', `\`gkill-readwrite-server.mjs\` | ${mcpRW} (${mcpReadOnly} read + ${mcpWriteOnly} write + 1 plugin)`)
  add('src/mcp/README.md', `ツール数（上の表の ${mcpRead} / ${mcpWrite} / ${mcpRW}）`)
  add('src/mcp/ABOUT_TEST.md', `ツール数（Read ${mcpRead} / Write ${mcpWrite} / ReadWrite ${mcpRW}）`)
  add('src/mcp/ABOUT_TEST.md', `Read サーバ ${mcpReadOnly} + プラグイン1 = ${mcpRead}ツール`)
  add('src/mcp/ABOUT_TEST.md', `Write サーバ ${mcpWrite - 1}（書き込み${mcpWriteOnly} + Read便利${mcpWriteConvenience}）+ プラグイン1 = ${mcpWrite}ツール`)
  add('src/mcp/ABOUT_TEST.md', `統合サーバ ${mcpRW - 1} + プラグイン1 = ${mcpRW}ツール`)
  add('src/mcp/ABOUT_TEST.md', `Read ${mcpReadOnly}ツール分のハンドラ実行ロジック`)
  add('src/mcp/ABOUT_TEST.md', `${mcpWrite}ツールディスパッチ`)
  add('src/mcp/ABOUT_TEST.md', `Write ${mcpWriteOnly}ツール定義（実物 import）`)
  add('src/mcp/ABOUT_TEST.md', `${mcpRW}ツール全ディスパッチ`)
  add('src/mcp/ABOUT_TEST.md', `Read ${mcpReadOnly}ツール + Write ${mcpWriteOnly}ツール`)

  // documents/reverse 側の MCP ツール数。2026-08-24 の監査で、検査対象が src/mcp と
  // スキル・マニュアルの4ファイルに限られていたため reverse 資料に3〜4世代前の数が
  // 14箇所残っていた（同一ファイル内で新旧が同居する自己矛盾も2件）。言及箇所を全て検査に載せる。
  add('documents/reverse/folder-structure.md', `${mcpRead}ツール = 固有${mcpReadOnly} + プラグイン1、port 8808`)
  add('documents/reverse/folder-structure.md', `${mcpWrite}ツール = 固有${mcpWrite - 1} + プラグイン1、port 8809`)
  add('documents/reverse/folder-structure.md', `${mcpRW}ツール = 固有${mcpRW - 1} + プラグイン1、port 8810`)
  add('documents/reverse/glossary.md', `${mcpRead}ツール = 固有${mcpReadOnly} + プラグイン1、stdio/HTTP`)
  add('documents/reverse/glossary.md', `${mcpWrite}ツール = 書き込み${mcpWriteOnly} + Read便利${mcpWriteConvenience} + プラグイン1、stdio/HTTP`)
  add('documents/reverse/glossary.md', `${mcpRW}ツール = 固有${mcpRW - 1} + プラグイン1、stdio/HTTP`)
  add('documents/reverse/mcp-setup-guide.md', `\`gkill-read-server.mjs\` | ${mcpRead} |`)
  add('documents/reverse/mcp-setup-guide.md', `\`gkill-write-server.mjs\` | ${mcpWrite} |`)
  add('documents/reverse/mcp-setup-guide.md', `\`gkill-readwrite-server.mjs\` | ${mcpRW} |`)
  add('documents/reverse/design-philosophy.md', `Read（${mcpRead}ツール、読み取りのみ）`)
  add('documents/reverse/design-philosophy.md', `Write（${mcpWrite}ツール）と ReadWrite（${mcpRW}ツール）`)
  add('documents/reverse/usecase.md', `Read サーバー（${mcpRead}ツール）は読み取りのみ、Write（${mcpWrite}ツール）/ ReadWrite（${mcpRW}ツール）`)
  add('documents/reverse/testing-guide.md', `Read ${mcpReadOnly}ツール分のハンドラ実行ロジック`)
  add('documents/reverse/testing-guide.md', `${mcpWrite}ツールディスパッチ`)
  add('documents/reverse/testing-guide.md', `Write ${mcpWriteOnly}ツール定義（実物 import）`)
  add('documents/reverse/testing-guide.md', `全${mcpRW}ツールディスパッチ`)
  add('documents/reverse/api-endpoints.md', `MCPツール${mcpRead}個`)

  // ── KFTL ステートメント型数 / glossary 用語数
  add('.claude/skills/gkill-client-kftl/SKILL.md', `KFTL parser (${m.kftlStatementTs} statement types; the Go side has ${m.kftlStatementGo})`)
  add('documents/reverse/folder-structure.md', `KFTLパーサー（${m.kftlStatementTs}ステートメント型）`)
  // frontend-architecture / glossary にも同じ数がある。folder-structure だけ検査していたため
  // 48/50 の割れが残っていた（2026-08-24 監査）。
  add('documents/reverse/frontend-architecture.md', `KFTLパーサー (${m.kftlStatementTs} ステートメント型`)
  add('documents/reverse/glossary.md', `（${m.kftlStatementTs}ステートメント型。\`kftl_*/\` 配下の具象クラス数）`)
  add('documents/reverse/folder-structure.md', `バックエンド側、${m.kftlStatementGo}ステートメント型`)
  add('.claude/skills/gkill-docs/SKILL.md', `glossary.md (${m.glossaryTerms} terms)`)
  add('documents/reverse/README.md', `ドメイン用語の定義（${m.glossaryTerms}項目）`)
  add('documents/reverse/folder-structure.md', `用語集（${m.glossaryTerms}項目）`)

  // ── docコメント網羅率。
  //    reps のインターフェース契約と HandleXxx は 100% 維持する方針なので、
  //    ここが落ちたら「docを書かずにメソッド/ハンドラを足した」ことを意味する。
  add('.claude/skills/gkill-go-backend/SKILL.md', `${m.repsIfaceDocumented}/${m.repsIfaceMethods} documented`)
  add('.claude/skills/gkill-go-backend/SKILL.md', `${m.handlerDocumented}/${m.handlerMethods} handlers documented`)

  // ── ユースケース数（ユニークUC-ID） / シーケンス図数（mermaidブロック）
  add('documents/reverse/usecase.md', `**${m.ucIds}件（ユニークな UC-ID 数）**`)
  add('documents/reverse/README.md', `ユースケース一覧（${m.ucIds}件）`)
  add('documents/reverse/scenario.md', `（${m.ucIds}件、1操作ずつ静的に列挙）`)
  add('documents/reverse/folder-structure.md', `ユースケース一覧（${m.ucIds}件）`)
  add('.claude/skills/gkill-docs/SKILL.md', `usecase.md (${m.ucIds} use cases)`)
  add('documents/reverse/README.md', `シーケンス図（${m.seqDiagrams}本:`)
  add('documents/reverse/folder-structure.md', `シーケンス図（${m.seqDiagrams}本:`)
  add('.claude/skills/gkill-docs/SKILL.md', `sequence-diagrams.md (${m.seqDiagrams} diagrams)`)

  // ── handle_*.go ファイル数（CLAUDE.md / サーバ系README）
  const handlerTests = m.handlers - m.handlersImpl
  add('.claude/skills/gkill-go-backend/SKILL.md', `HTTP API handlers (${m.handlers} files incl. tests, 1 handler per file)`)
  add('.claude/skills/gkill-go-backend/SKILL.md', `repositories.WriteThroughXxxCache(ctx, ...)\` を使うこと（${m.writeThroughCalls}箇所）`)
  add('src/server/README.md', `（${m.handlers} handle_*.go`)
  add('src/server/gkill/api/README.md', `handle_*.go は${m.handlers}ファイル（実装${m.handlersImpl} + テスト${handlerTests}）`)
  add('src/server/gkill/api/gkill_server_api/README.md', `実装${m.handlersImpl}ファイル + テスト${handlerTests}ファイル`)

  // ── エラー/メッセージコード数 / FindQueryフィールド数 / Goバージョン
  add('src/server/gkill/api/README.md', `エラーコード定数（${m.errCodes}定数`)
  add('src/server/gkill/api/README.md', `メッセージコード定数（${m.msgCodes}定数`)
  add('src/server/gkill/api/message/README.md', `（${m.errCodes} 定数:`)
  add('src/server/gkill/api/message/README.md', `（${m.msgCodes} 定数:`)
  add('src/server/gkill/api/message/README.md', `## エラーコード体系（${m.errCodes} コード）`)
  add('src/server/gkill/api/find/README.md', `${m.findQueryFields} フィールドの検索条件`)
  add('src/server/gkill/api/README.md', `（${m.findQueryFields}フィールド:`)
  add('src/server/README.md', `**Go バージョン**: ${m.goVersion}`)
  add('AGENTS.md', `declares \`go ${m.goVersion}\``)

  // ── ADR 件数。documents/adr/ 配下の NNNN-*.md（0000-template.md を除く）。
  //   ADR は増える一方なので、CLAUDE.md の件数だけが古びるのを防ぐ。
  add('.claude/skills/gkill-docs/SKILL.md', `Architecture Decision Record（現在 ${m.adrCount} 件）`)

  // ── テストファイル数（ディレクトリ単位） / Wear OS Kotlinファイル数
  add('src/server/ABOUT_TEST.md', `リポジトリ実装 (${m.repsTestFiles}ファイル)`)
  add('src/ABOUT_TEST.md', `${m.datasTestFiles}ファイル（データモデル + 横断検証）`)
  add('src/client/ABOUT_TEST.md', `データモデル (${m.datasTestFiles}ファイル`)
  add('src/client/classes/datas/ABOUT_TEST.md', `テストする（${m.datasTestFiles}ファイル）`)
  add('src/client/ABOUT_TEST.md', `ユニットテスト（${m.unitTests}テスト宣言、${m.unitTestFiles}ファイル）`)
  add('src/client/ABOUT_TEST.md', `合計${m.unitTests + m.e2eTests}テスト宣言`)
  add('src/locales/ABOUT_TEST.md', `一致すること（${m.i18nKeys}キー）`)
  add('src/client/README.md', `View コンポーネント（${m.views} .vue）`)
  add('src/client/README.md', `Dialog コンポーネント（${m.dialogs} .vue）`)
  add('src/client/pages/README.md', `View コンポーネント（${m.views} .vue）`)
  add('src/client/pages/README.md', `Dialog コンポーネント（${m.dialogs} .vue）`)
  add('src/wear_os/README.md', `Kotlin ソース（${m.wearCompanionKt}ファイル）`)
  add('src/wear_os/README.md', `Kotlin ソース（${m.wearWatchKt}ファイル）`)

  // ── 2026-08 追加分。
  //   ここから下は「同じ数字が複数の資料に散っているのに、検査は1〜2ファイルしか
  //   見ていなかった」ために静かにドリフトしていた箇所。実例:
  //     - req_res が api-endpoints.md だけ 182 のまま（他は186）
  //     - gkill-api.ts の行数が 3,330 と 3,660 の2説に割れていた
  //     - エラーコードが「406定数」なのに範囲は「〜ERR000401」のまま
  //   数字を書いた資料は漏れなくここに登録すること。

  // ── 2026-08-10 追加分。
  //   検査から漏れていたために、同じ数字が資料の中で自己矛盾していた箇所。実例:
  //     - folder-structure.md のダイアログ数だけ 103 のまま（他は110）
  //     - pages/ABOUT_TEST.md が冒頭「41 specファイル218テスト宣言」・
  //       見出し「38 specファイル, 212テスト宣言」と同一ファイル内で食い違い
  //     - pages/ABOUT_TEST.md の Composable ファイル数が 18 のまま（実測と乖離）
  add('documents/reverse/folder-structure.md', `ダイアログコンポーネント（${m.dialogs}ファイル`)
  add('src/client/pages/ABOUT_TEST.md', `${m.e2eTestFiles} specファイル${m.e2eTests}テスト宣言`)
  add('src/client/pages/ABOUT_TEST.md', `### E2E テスト（${m.e2eTestFiles} specファイル, ${m.e2eTests}テスト宣言）`)
  add('src/client/pages/ABOUT_TEST.md', `### Composable ユニットテスト（${m.unitComposablesFiles}ファイル）`)
  // e2eTestFiles は *.spec.ts の数（auth.setup.ts は含まない）
  add('documents/reverse/testing-guide.md', `| フロントエンド E2E | ${m.e2eTests} | ${m.e2eTestFiles}（+auth.setup.ts） |`)

  // ── 2026-08-20 追加分。
  //   ここから下も「同じ数字が複数の資料に散っているのに、検査は1〜2ファイルしか
  //   見ていなかった」ぶん。今回の点検で実際にずれていたものだけを登録してある。
  //     - CLAUDE.md のビュー数だけ 203 のまま（ダイアログ数は検査済みで正しかった）
  //     - screen-specs.md の合計行の**括弧の中**だけ 203 で、203+116+15=334≠333
  //     - documents/reverse/README.md の画面仕様だけ 307 のまま（実測 333）
  //     - testing-guide.md の本文だけ「40 specファイル・215テスト宣言」のまま
  //     - src/ABOUT_TEST.md の索引表だけ 894 / 719 / 18 / 1342 のまま
  //     - マニュアルのページ数が 21 のまま（ポート画面の追加で 22）
  //     - ルート数が「13」「14」「表14行」で三重に食い違い

  // ビュー数（ダイアログ数は上で検査済み。CLAUDE.md はビューだけ漏れていた）
  add('.claude/skills/gkill-client-foundation/SKILL.md', `${m.views} view components`)
  // 合計行の内訳。合計だけ検査していると括弧の中が置き去りになる
  add('documents/reverse/screen-specs.md',
    `ビュー${m.views} + ダイアログ${m.dialogs} + ページ${m.pages}`)
  add('documents/reverse/README.md', `画面仕様・項目定義（${m.componentTotal}コンポーネント）`)

  // ルート数（コンポーネントを持つルート / redirect 専用ルート）
  const routeTotal = m.routeComponents + m.routeRedirects
  add('.claude/skills/gkill-client-foundation/SKILL.md',
    `${m.routeComponents} page routes`)
  add('.claude/skills/gkill-client-foundation/SKILL.md', `${m.routeRedirects} redirect-only routes`)
  add('documents/reverse/screen-specs.md', `| ルート | ${m.routeComponents} |`)
  add('documents/reverse/screen-specs.md',
    `で定義される${routeTotal}ルート（コンポーネント${m.routeComponents} + リダイレクト専用${m.routeRedirects}）`)
  add('documents/reverse/screen-transition.md',
    `### ルートページ一覧（${m.routeComponents}ルート）`)
  add('documents/reverse/program-spec.md', `### ルート構成（${m.routeComponents}ルート）`)
  add('documents/reverse/glossary.md', `${m.routeComponents}ルートの定義`)
  add('documents/reverse/folder-structure.md', `ルーター定義（${m.routeComponents}ルート）`)
  add('src/client/README.md', `## ルーティング（${m.routeComponents}ルート）`)
  add('src/client/ABOUT_TEST.md', `ルーター (${m.routeComponents}ルート`)

  // マニュアルのページ数（言語あたり）
  add('.claude/skills/gkill-docs/SKILL.md', `${m.manualPages} pages per language`)
  add('documents/reverse/folder-structure.md', `7言語×${m.manualPages}ページ`)

  // src/ABOUT_TEST.md の索引表（上の統計表とは別に、各行が件数を書いている）
  add('src/ABOUT_TEST.md', `フロントエンド全体（unit ${m.unitTests} + E2E ${m.e2eTests}）`)
  add('src/ABOUT_TEST.md', `Go バックエンド全体（${m.goTests}テスト / ${m.goTestPkgs}パッケージ）`)
  add('src/ABOUT_TEST.md', `MCP サーバ（${m.mcpTests}テスト）`)
  add('src/ABOUT_TEST.md', `${m.sdkTests}テスト）`)

  // src/client/ABOUT_TEST.md のツリー内の件数（冒頭の要約とは別の箇所）
  add('src/client/ABOUT_TEST.md', `# ユニットテスト (${m.unitTests}テスト宣言, ${m.unitTestFiles}ファイル)`)
  add('src/client/ABOUT_TEST.md', `# E2E テスト (${m.e2eTests}テスト宣言, ${m.e2eTestFiles} specファイル`)
  add('src/client/ABOUT_TEST.md', `### E2E テスト (${m.e2eTests}テスト宣言, ${m.e2eTestFiles} specファイル)`)
  add('src/client/ABOUT_TEST.md', `D-note モジュール (${m.unitDnoteFiles}ファイル`)
  add('src/client/ABOUT_TEST.md', `| D-note | ${m.unitDnoteFiles}ファイル |`)

  // testing-guide.md の本文（先頭の表とは別の箇所）
  add('documents/reverse/testing-guide.md',
    `${m.e2eTestFiles} specファイル + auth.setup.ts、${m.e2eTests}テスト宣言`)

  // src/server/ABOUT_TEST.md のカテゴリ表。合計行が「verify_docs と一致する」と
  // 自称していたのに検査されておらず、実測125に対して78のまま放置されていた
  add('src/server/ABOUT_TEST.md', `**合計 ${m.goTestFiles} ファイル**`)
  add('src/server/ABOUT_TEST.md', `| API 統合 | ${m.serverApiTestFiles} |`)
  add('src/server/ABOUT_TEST.md', `| リポジトリ | ${m.repsAllTestFiles} |`)
  add('src/server/ABOUT_TEST.md', `| プラグイン SDK | ${m.sdkTestFiles} |`)

  // go.mod のモジュール数（依存を上げるとき全モジュールで tidy が要る）
  add('src/tools/README.md', `全 go.mod（${m.goModModules}モジュール自動発見）`)

  // gkill-api.ts の行数（100の位で丸め。「3,400」表記）
  const apiLines = m.gkillApiLinesApprox.toLocaleString('en-US')
  add('documents/reverse/frontend-architecture.md', `(~${apiLines}行)`)
  add('documents/reverse/frontend-architecture.md', `約${apiLines}行`)
  add('documents/reverse/folder-structure.md', `~${apiLines}行`)
  add('documents/reverse/glossary.md', `約${apiLines}行`)
  add('documents/reverse/program-spec.md', `約${apiLines}行`)
  add('documents/reverse/class-diagrams.md', `約${apiLines}行`)
  add('.claude/skills/gkill-client-foundation/SKILL.md', `(~${apiLines} lines)`)

  // req_res（Go / TypeScript）
  add('documents/reverse/api-endpoints.md', `req_res/\`（${m.reqRes}ファイル）`)
  add('documents/reverse/api-endpoints.md', `構造体（${m.reqRes}ファイル）`)
  add('.claude/skills/gkill-go-backend/SKILL.md', `Request/response structs for every endpoint (${m.reqRes} files)`)
  add('documents/reverse/folder-structure.md', `リクエスト/レスポンス型（${m.classesApiReqRes}ファイル）`)
  add('documents/reverse/frontend-architecture.md',
    `(${m.classesApiReqRes}ファイル、サーバー側は${m.reqRes}ファイル)`)
  add('documents/reverse/glossary.md', `TypeScript 版入出力型（${m.classesApiReqRes}ファイル）`)

  // エンドポイント: ルート表の行数が唯一の数（定義 = 登録）。内訳（POST/GET・認証区分）も表から数える
  add('documents/reverse/api-endpoints.md',
    `エンドポイント ${m.endpoints}件（${m.endpointsPost} POST + ${m.endpointsGet} GET）`)
  add('documents/reverse/README.md', `全${m.endpoints}エンドポイントのリファレンス`)
  add('documents/reverse/README.md', `全APIエンドポイントのリファレンス（${m.endpoints}件）`)
  add('documents/reverse/folder-structure.md', `APIエンドポイント一覧（${m.endpoints}件）`)
  add('documents/reverse/folder-structure.md',
    `ルート表（${m.endpoints}エンドポイント: ${m.endpointsPost} POST + ${m.endpointsGet} GET`)
  add('documents/reverse/glossary.md', `全${m.endpoints}エンドポイントのパス・メソッド・認証区分・ハンドラ`)
  add('documents/reverse/program-spec.md', `全${m.endpoints}エンドポイント（${m.endpointsPost} POST + ${m.endpointsGet} GET）のハンドリング`)
  add('documents/reverse/program-spec.md',
    `### エンドポイント分類（${m.endpoints}件 = ${m.endpointsPost} POST + ${m.endpointsGet} GET`)
  add('documents/reverse/program-spec.md', `ルート表で定義されます（${m.endpoints}件`)
  add('src/server/gkill/api/README.md', `## 全エンドポイント一覧（${m.endpoints}エンドポイント）`)
  add('documents/reverse/program-spec.md', `| \`wrapNoAuth\` | ${m.wrapNoAuth} |`)
  add('documents/reverse/program-spec.md', `| \`wrapAuth\` | ${m.wrapAuth} |`)
  add('documents/reverse/program-spec.md', `| \`wrapAuthRepos\` | ${m.wrapAuthRepos} |`)

  // エラー/メッセージコードは件数だけでなく採番の上端も見る。
  // 「406定数、ERR000001〜ERR000401」のように件数だけ直された状態を弾くため。
  add('documents/reverse/error-handling-and-security.md',
    `合計 **${m.errCodes}件** のエラーコードが存在する（ERR000001〜${m.errCodeMax}`)
  add('documents/reverse/glossary.md', `ERR000001〜${m.errCodeMax} の定数定義（計${m.errCodes}件`)
  add('src/server/gkill/api/message/README.md', `\`${m.errCodeMax}\` |`)
  add('src/server/gkill/api/message/README.md', `\`${m.msgCodeMax}\` |`)
  add('src/server/gkill/api/README.md', `ERR000001〜${m.errCodeMax}`)
  add('documents/reverse/error-handling-and-security.md',
    `\`ERR900001\`〜\`${m.clientErrCodeMax}\` の${m.clientErrCodes}件`)

  // i18n / ディレクトリ別ファイル数
  add('documents/reverse/glossary.md', `（${m.i18nKeys}キー/言語）`)
  add('documents/reverse/folder-structure.md', `HTTPハンドラ層（${m.serverApiGo}ファイル）`)
  add('documents/reverse/folder-structure.md', `Composition関数群（${m.classesUse}ファイル）`)
  add('documents/reverse/frontend-architecture.md', `等、${m.classesUse}ファイル）`)
  add('documents/reverse/frontend-architecture.md', `共有ロジック（${m.classesUse}ファイル）`)

  // テスト件数（合計行 / ディレクトリ別）。合計は手計算で放置されやすいので実測から出す。
  const totalRow = `| **合計** | **${m.totalTests.toLocaleString('en-US')}** | **${m.totalTestFiles}** |`
  add('src/ABOUT_TEST.md', totalRow)
  add('documents/reverse/testing-guide.md', totalRow)
  add('documents/reverse/testing-guide.md', `ハンドラ層（${m.serverApiTestFiles}ファイル）`)
  add('documents/reverse/testing-guide.md', `リポジトリ実装（${m.repsTestFiles}ファイル`)
  add('documents/reverse/testing-guide.md', `CLI・エントリポイント（${m.serverMainTestFiles}ファイル）`)
  add('documents/reverse/testing-guide.md', `ユーティリティ（${m.unitClassesFiles}ファイル）`)
  add('documents/reverse/testing-guide.md', `Vue Composable（${m.unitComposablesFiles}ファイル`)
  add('src/client/ABOUT_TEST.md', `ユーティリティクラス (${m.unitClassesFiles}ファイル`)
  add('src/client/ABOUT_TEST.md', `Vue Composable (${m.unitComposablesFiles}ファイル`)
  add('src/client/ABOUT_TEST.md', `| ユーティリティ | ${m.unitClassesFiles}ファイル |`)
  add('src/client/ABOUT_TEST.md', `| Composable | ${m.unitComposablesFiles}ファイル |`)
  add('src/client/pages/ABOUT_TEST.md', `${m.e2eTests}テスト宣言`)

  return A
}

function checkCounts(m) {
  // ロケール間のキー数一致
  const counts = Object.entries(m.localeKeyCounts)
  const mismatched = counts.filter(([, c]) => c !== m.i18nKeys)
  if (mismatched.length) {
    err(`i18n キー数がロケール間で不一致: ja=${m.i18nKeys}, ` +
      mismatched.map(([f, c]) => `${f}=${c}`).join(', '))
  }

  for (const { file, phrase } of buildCountAssertions(m)) {
    if (!exists(file)) { err(`件数検査: ファイルが存在しない: ${file}`); continue }
    const text = readText(file)
    if (!text.includes(phrase)) {
      err(`件数ドリフト: ${file} に期待語句が見つからない → 「${phrase}」（実測に合わせて更新が必要）`)
    }
  }
}

// ─────────────────────────────────────────────────────────────
// 3. 相互リンク（reverse docs の .md リンクが実在するか）
// ─────────────────────────────────────────────────────────────
// コードフェンスの中を落とす。
//
// **バッククォート対の走査は必ずこれを通すこと。** フェンスは3連バッククォートなので、
// 生のテキストへ「` ... `」の正規表現を当てると
// 「開きフェンスの3本目」と「閉じフェンスの1本目」が対になり、以降の対応が全部ずれる。
// 実測では223個のパス記述のうち78個しか見えていなかった
// （frontend-architecture.md は528スパン中0個）。
function stripFencedBlocks(text) {
  return text.replace(/^\s*```[\s\S]*?^\s*```/gm, '\n')
}

// ─────────────────────────────────────────────────────────────
// 検査対象の Markdown（リポジトリルートからの相対パス）。
//   documents/reverse/*.md に加え、README.md 群と ABOUT_TEST.md 群も対象にする。
//   src/README.md の「各サブディレクトリの README / ABOUT_TEST」表がリンク切れのまま
//   放置される事故を防ぐのが主目的。
//
//   **検査対象の資料ジャンルを増やすときは、必ずこの関数へ足すこと。**
//   ここが唯一の入口なので、足し忘れるとリンク切れもゴーストファイル名も素通りする
//   （実例: documents/releasenote/ の27ファイルは今も対象外）。
//   AGENTS.md と .claude/skills/**/SKILL.md も対象（AI 資料再編。ADR-0804）。
function docMarkdownFiles() {
  const out = []
  for (const f of listFiles('documents/reverse', (f) => f.endsWith('.md'))) {
    out.push('documents/reverse/' + f)
  }
  // ADR。0000-template.md はプレースホルダを含むので対象外。
  for (const f of listFiles('documents/adr', (f) => f.endsWith('.md') && f !== '0000-template.md')) {
    out.push('documents/adr/' + f)
  }
  if (exists('README.md')) out.push('README.md')
  if (exists('CLAUDE.md')) out.push('CLAUDE.md')
  if (exists('AGENTS.md')) out.push('AGENTS.md')
  // 規約スキル（領域別の不変条件の正本）。SKILL.md すべてを検査対象にする。
  // .gitignore が /.claude/* + !/.claude/skills/ である前提（checkSkills が0件を error にする）。
  for (const p of listFilesRec('.claude/skills', (f) => f.endsWith('.md'))) {
    out.push(path.relative(ROOT, p).split(path.sep).join('/'))
  }
  for (const p of listFilesRec('src', (f) => f === 'README.md' || f === 'ABOUT_TEST.md')) {
    out.push(path.relative(ROOT, p).split(path.sep).join('/'))
  }
  return out
}

function checkLinks() {
  const linkRe = /\]\(([^)]+)\)/g
  for (const rel of docMarkdownFiles()) {
    const dir = path.dirname(rel)
    const text = readText(rel)
    let mt
    while ((mt = linkRe.exec(text)) !== null) {
      const target = mt[1].trim()
      if (/^(https?:)?\/\//.test(target) || target.startsWith('#') || target.startsWith('mailto:')) continue
      const hashIdx = target.indexOf('#')
      const filePart = hashIdx >= 0 ? target.slice(0, hashIdx) : target
      if (!filePart) continue // 同一ファイル内アンカー
      const resolved = path.join(dir, filePart)
      if (!exists(resolved)) {
        err(`リンク切れ: ${rel} → ${target}`)
      }
    }
  }
}

// ─────────────────────────────────────────────────────────────
// 4. 参照パス（docs 内の `src/...` 等の実在確認）※警告のみ
//    glob（*）や説明用の一般パスは除外し、具体ファイル/ディレクトリのみ検査。
// ─────────────────────────────────────────────────────────────
function checkPaths() {
  // 対象は docMarkdownFiles() と同じ（README / ABOUT_TEST 群も含む）。
  // 以前は documents/reverse + CLAUDE.md だけだったので、
  // src/**/README.md のパス記述はまったく検査されていなかった。
  const mdFiles = docMarkdownFiles()

  const codeRe = /`([^`\n]+)`/g
  const seen = new Set()
  for (const rel of mdFiles) {
    const text = stripFencedBlocks(readText(rel))
    let mt
    while ((mt = codeRe.exec(text)) !== null) {
      const tok = mt[1].trim()
      if (!/^(src|documents|resources|public)\/[\w./-]+$/.test(tok)) continue
      if (tok.includes('*')) continue
      // 拡張子を持つ、あるいは末尾がディレクトリ名らしいものだけを対象
      if (!/\.\w+$/.test(tok) && !tok.endsWith('/')) continue
      const key = rel + '::' + tok
      if (seen.has(key)) continue
      seen.add(key)
      if (!exists(tok)) {
        warn(`参照パス未検出（要確認）: ${rel} → ${tok}`)
      }
    }
  }
}

// ─────────────────────────────────────────────────────────────
// 4-b. 資料に載っているファイル名が実在するか
//
//   件数だけを検査していると「数は合っているのに一覧は古い」が通り抜ける。
//   実例: classes/dnote/README.md は `dnote-predicate/（31ファイル）` の
//   件数検査を通ったまま、削除済みの述語2件を表に載せ続けていた。
//   ASCII ツリーや表セルに書かれた**素のファイル名**（バッククォートの有無を問わない）を
//   拾い、同名のファイルがリポジトリのどこにも無ければ落とす。
//
//   パスではなくファイル名で照合するのは、資料の多くがツリー図で
//   「置き場所は図の形で示し、行にはファイル名だけ書く」形式だから。
//   同名別ディレクトリを見逃す代わりに、削除・改名の取り残しを確実に捕まえる。
// ─────────────────────────────────────────────────────────────
const DOC_FILENAME_EXTENSIONS = ['go', 'ts', 'vue', 'mjs', 'kt']
// 走査から外すディレクトリ（生成物・依存・IDE）
const DOC_FILENAME_SKIP_DIRS = new Set([
  'node_modules', '.git', 'dist', 'release', 'build', '.gradle', '.idea',
  'test-results', 'playwright-report',
])
// 実在を求めない書き方。
//   - `_repository.go` のような**接尾辞パターン**（先頭が `_`）
//   - `xxx_dao.go` / `use-xxx-view.ts` のような**プレースホルダ**
// どちらも資料が意図して使っている記法なので、実ファイルを探しに行かない。
const DOC_FILENAME_PLACEHOLDER = /^_|xxx|yyy|zzz/

function collectRepositoryBasenames() {
  const names = new Set()
  const walk = (dir) => {
    for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
      if (entry.isDirectory()) {
        if (DOC_FILENAME_SKIP_DIRS.has(entry.name)) continue
        walk(path.join(dir, entry.name))
        continue
      }
      names.add(entry.name)
    }
  }
  walk(ROOT)
  return names
}

function checkDocFilenames() {
  const basenames = collectRepositoryBasenames()
  const fileRe = new RegExp(
    `(?<![\\w./-])([A-Za-z0-9_][\\w.-]*\\.(?:${DOC_FILENAME_EXTENSIONS.join('|')}))(?![\\w-])`, 'g')

  // 同じ名前を何度も報告しない（ツリーと表で二重に出るため）
  const missing = new Map()
  for (const rel of docMarkdownFiles()) {
    // ADR は本質的に歴史を語るので、削除済みファイルの名前が本文に出てくる。
    // 「かつて存在した名前はコードフェンスで囲む」を規約にして、フェンス内だけ免除する
    // （documents/adr/README.md「削除済みファイルの名前を書くとき」）。
    // フェンスの外は従来どおり実在を要求するので、現役ファイルの改名取り残しは捕まる。
    const text = rel.startsWith('documents/adr/') ? stripFencedBlocks(readText(rel)) : readText(rel)
    for (const mt of text.matchAll(fileRe)) {
      const name = mt[1]
      if (DOC_FILENAME_PLACEHOLDER.test(name)) continue
      if (basenames.has(name)) continue
      if (!missing.has(name)) missing.set(name, new Set())
      missing.get(name).add(rel)
    }
  }
  for (const name of [...missing.keys()].sort()) {
    err(`資料に載っているファイルが実在しない: ${name}`
      + `（${[...missing.get(name)].sort().join(', ')}）`)
  }
}

// ─────────────────────────────────────────────────────────────
// 5. Mermaid ブロックの軽量検証（構文の型宣言があるか）
// ─────────────────────────────────────────────────────────────
function checkMermaid() {
  const dir = 'documents/reverse'
  const mds = listFiles(dir, (f) => f.endsWith('.md'))
  const known = ['graph', 'flowchart', 'sequenceDiagram', 'classDiagram',
    'stateDiagram', 'stateDiagram-v2', 'erDiagram', 'journey', 'gantt',
    'pie', 'gitGraph', 'mindmap', 'timeline', 'quadrantChart']
  const blockRe = /```mermaid\r?\n([\s\S]*?)```/g
  for (const f of mds) {
    const rel = path.join(dir, f)
    const text = readText(rel)
    let mt
    let idx = 0
    while ((mt = blockRe.exec(text)) !== null) {
      idx++
      const body = mt[1].trim()
      if (!body) { err(`Mermaid 空ブロック: ${rel} #${idx}`); continue }
      const firstLine = body.split(/\r?\n/)[0].trim()
      const ok = known.some((k) => firstLine.startsWith(k))
      if (!ok) {
        warn(`Mermaid 図種別が不明（要確認）: ${rel} #${idx} → 「${firstLine.slice(0, 40)}」`)
      }
    }
  }
}

// ─────────────────────────────────────────────────────────────
// 6. マニュアル（resources/manual）の検証
//    - 生成鮮度: manual_src から生成した内容が on-disk と一致するか
//    - a11y 不変条件: <main> 有り / 全 <table> に <caption> / 素の <th> 無し
//    - 言語構成一致: 全言語のページ集合が同一
//    - マニュアル内リンク: 相対リンクが解決するか
// ─────────────────────────────────────────────────────────────
function checkManuals() {
  // 生成鮮度
  let rendered
  try {
    rendered = renderAll()
  } catch (e) {
    err(`マニュアル生成に失敗: ${e.message}`)
    return
  }
  for (const { relPath, outPath, content } of rendered) {
    if (!fs.existsSync(outPath)) { err(`マニュアル未生成: ${relPath}（npm run build_manuals）`); continue }
    // 改行差（CRLF/LF）は無視して比較（LFは .gitattributes で固定）
    if (fs.readFileSync(outPath, 'utf8').replace(/\r\n/g, '\n') !== content.replace(/\r\n/g, '\n')) {
      err(`マニュアル生成鮮度NG: ${relPath} が manual_src と不一致（npm run build_manuals で再生成）`)
    }
  }

  if (!fs.existsSync(OUT_DIR)) { err('resources/manual が存在しない'); return }
  const langs = fs.readdirSync(OUT_DIR).filter((d) =>
    fs.statSync(path.join(OUT_DIR, d)).isDirectory())

  // 言語構成一致
  let baseSet = null
  let baseLang = null
  for (const lang of langs) {
    const pages = fs.readdirSync(path.join(OUT_DIR, lang)).filter((f) => f.endsWith('.html')).sort()
    if (baseSet === null) { baseSet = pages.join(','); baseLang = lang; continue }
    if (pages.join(',') !== baseSet) {
      err(`マニュアル言語構成の不一致: ${lang} のページ集合が ${baseLang} と異なる`)
    }
  }

  // a11y 不変条件 + マニュアル内リンク
  for (const lang of langs) {
    const langDir = path.join(OUT_DIR, lang)
    for (const page of fs.readdirSync(langDir).filter((f) => f.endsWith('.html'))) {
      const rel = `resources/manual/${lang}/${page}`
      const text = fs.readFileSync(path.join(langDir, page), 'utf8')
      if (!text.includes('<main>')) err(`a11y: <main> 無し: ${rel}`)
      const tables = (text.match(/<table>/g) || []).length
      const captions = (text.match(/<caption/g) || []).length
      if (tables !== captions) err(`a11y: <table>(${tables}) と <caption>(${captions}) の数が不一致: ${rel}`)
      if (/<th>/.test(text)) err(`a11y: scope 無しの <th> が残存: ${rel}`)

      // 相対リンク解決（href="...")
      const hrefRe = /href="([^"]+)"/g
      let mt
      while ((mt = hrefRe.exec(text)) !== null) {
        let href = mt[1].trim()
        if (/^(https?:)?\/\//.test(href) || href.startsWith('#') || href.startsWith('mailto:')) continue
        const q = href.indexOf('?')
        const filePart = q >= 0 ? href.slice(0, q) : href
        if (!filePart) continue
        if (!fs.existsSync(path.join(langDir, filePart))) {
          err(`マニュアル内リンク切れ: ${rel} → ${href}`)
        }
      }
    }
  }
}

// ─────────────────────────────────────────────────────────────
// 6-b. ユーザー向け資料の用語検査
//   方針: ユーザー向け資料は開発コード名を出さず、UI（i18n）に出てくる語だけを使う。
//   ここでは「UI ラベルとして使われていない開発コード名」を明示的に列挙して禁止する。
//   ロケール値に出るかどうかで自動判定はしない —— NOT_FOUND_*_ERROR_MESSAGE が
//   Kmemo / URLog / ReKyou 等をそのまま出しており、それを根拠にすると素通りしてしまうため。
// ─────────────────────────────────────────────────────────────
const USER_DOC_FORBIDDEN_TERMS = [
  'IDFKyou', 'IDF', 'WAN', 'MiReKyou', 'ReKyou', 'Kyou', 'KFTL',
  'Rykv', 'Mkfl', 'Dnote', 'Ryuu', 'Playing', 'Lantana', 'Nlog',
  'URLog', 'TimeIs', 'Kmemo', 'DVNF', 'RepType',
  // ポート画面の開発コード名。URL（/rudbeckia）・ファイル名・保存キーには出るが、
  // 利用者に見せる呼び名は「ポート」だけ。現状マニュアル本文に漏れは無く、
  // 止め金がこれしか無いので入れておく
  'Rudbeckia', 'rudbeckia',
]
// 例外: Saihate は ja 以外の SAIHATE_APP_NAME がそのまま "Saihate" なので UI ラベル。
// `<code>` の中（server-config.html の rep type 一覧など、UI が生値を表示する箇所）は対象外。

function checkUserDocTerms(body, rel) {
  for (const term of USER_DOC_FORBIDDEN_TERMS) {
    if (new RegExp(`(^|[^A-Za-z])${term}([^A-Za-z]|$)`).test(body)) {
      err(`ユーザー向け資料用語NG: ${rel} に開発コード名「${term}」`
        + '（i18n の UI 用語に置き換えること）')
    }
  }
}

function checkManualTerminology() {
  if (!fs.existsSync(SRC_DIR)) return
  const langs = fs.readdirSync(SRC_DIR).filter((d) =>
    fs.statSync(path.join(SRC_DIR, d)).isDirectory())
  for (const lang of langs) {
    const langDir = path.join(SRC_DIR, lang)
    for (const page of fs.readdirSync(langDir).filter((f) => f.endsWith('.html'))) {
      // 属性付き（`<code class="...">`）も剥がす。`<code>` 完全一致で書いていると、
      // 属性を1つ足しただけで**遮蔽が外れて用語が露出する**のに検査は素通りする。
      //
      // `href` / `src` の値も落とす。ページのファイル名は開発コード名のままでよく
      // （`rudbeckia.html`）、利用者の目に入るのはリンクの**文字**のほうだから。
      // `alt` / `title` は読み上げやツールチップとして見えるので残す。
      const body = fs.readFileSync(path.join(langDir, page), 'utf8')
        .replace(/<code(?:\s[^>]*)?>[\s\S]*?<\/code>/g, '')
        .replace(/<!--[\s\S]*?-->/g, '')
        .replace(/\s(?:href|src)\s*=\s*"[^"]*"/g, '')
        .replace(/\s(?:href|src)\s*=\s*'[^']*'/g, '')
      checkUserDocTerms(body, `resources/manual_src/${lang}/${page}`)
    }
  }

  const userGuidePath = path.join(ROOT, 'documents/reverse/user-guide.md')
  if (!fs.existsSync(userGuidePath)) return
  // コードフェンスとインラインコードは運用コマンド等の正確な表記に必要なので除外する。
  // Markdownリンクは利用者に見えるラベルだけを残し、リンク先のファイル名・URLは検査しない。
  const userGuideBody = fs.readFileSync(userGuidePath, 'utf8')
    .replace(/```[\s\S]*?```/g, '')
    .replace(/`[^`\n]*`/g, '')
    .replace(/<!--[\s\S]*?-->/g, '')
    .replace(/\[([^\]]*)\]\([^)]+\)/g, '$1')
  checkUserDocTerms(userGuideBody, 'documents/reverse/user-guide.md')
}

// ─────────────────────────────────────────────────────────────
// 6-c. アプリから参照されるマニュアルページが実在するか
//   HelpDialog の screen_name と、直接 URL を組み立てている箇所を集める。
// ─────────────────────────────────────────────────────────────
function checkManualHelpPages() {
  const wanted = new Set()
  for (const p of listFilesRec('src/client', (f) => f.endsWith('.vue') || f.endsWith('.ts'))) {
    const t = fs.readFileSync(p, 'utf8')
    for (const m of t.matchAll(/screen_name\s*[=:]\s*["']([a-z0-9-]+)["']/g)) wanted.add(m[1])
    for (const m of t.matchAll(/\/resources\/manual\/\$\{[^}]+\}\/([a-z0-9-]+)\.html/g)) wanted.add(m[1])
  }
  const jaDir = path.join(SRC_DIR, 'ja')
  if (!fs.existsSync(jaDir)) return
  const have = new Set(fs.readdirSync(jaDir).filter((f) => f.endsWith('.html'))
    .map((f) => f.replace(/\.html$/, '')))
  for (const name of [...wanted].sort()) {
    if (!have.has(name)) {
      err(`アプリが参照するマニュアルページが無い: ${name}.html（HelpDialog screen_name / 直接URL）`)
    }
  }
}

// ─────────────────────────────────────────────────────────────
// 7. 構造パリティ（日本語=正本 に対する各言語の見出し/表構造のズレ）
//    翻訳・構造同期（Phase 5）の指針。通常は警告、--parity で詳細マトリクス。
// ─────────────────────────────────────────────────────────────
const CANONICAL_LANG = 'ja'

function pageShape(lang, page) {
  const p = path.join(SRC_DIR, lang, page)
  if (!fs.existsSync(p)) return null
  const t = fs.readFileSync(p, 'utf8')
  return {
    h2: (t.match(/<h2[\s>]/g) || []).length,
    h3: (t.match(/<h3[\s>]/g) || []).length,
    table: (t.match(/<table[\s>]/g) || []).length,
  }
}

function checkManualParity(detailed) {
  const jaDir = path.join(SRC_DIR, CANONICAL_LANG)
  if (!fs.existsSync(jaDir)) return
  const pages = fs.readdirSync(jaDir).filter((f) => f.endsWith('.html')).sort()
  const langs = fs.readdirSync(SRC_DIR)
    .filter((d) => fs.statSync(path.join(SRC_DIR, d)).isDirectory() && d !== CANONICAL_LANG)
    .sort()

  const rows = []
  for (const page of pages) {
    const ja = pageShape(CANONICAL_LANG, page)
    const diffs = []
    for (const lang of langs) {
      const s = pageShape(lang, page)
      if (!s) { diffs.push(`${lang}:欠落`); continue }
      const parts = []
      if (s.h2 !== ja.h2) parts.push(`h2=${s.h2}`)
      if (s.h3 !== ja.h3) parts.push(`h3=${s.h3}`)
      if (s.table !== ja.table) parts.push(`表=${s.table}`)
      if (parts.length) diffs.push(`${lang}(${parts.join(',')})`)
    }
    if (diffs.length) {
      rows.push({ page, ja, diffs })
      warn(`構造パリティ差異: ${page} [ja h2=${ja.h2},h3=${ja.h3},表=${ja.table}] ↔ ${diffs.join(' ')}`)
    }
  }

  if (detailed) {
    console.log(`\n構造パリティ（正本=${CANONICAL_LANG}、差異のあるページのみ）:`)
    if (!rows.length) { console.log('  （全ページ一致）'); return }
    for (const { page, ja, diffs } of rows) {
      console.log(`  ${page.padEnd(28)} ja[h2=${ja.h2} h3=${ja.h3} 表=${ja.table}]  → ${diffs.join('  ')}`)
    }
  }
}

// ─────────────────────────────────────────────────────────────
// 4-d. ADR（documents/adr/）の構造検査
//
//   ADR が持つのは「却下案・実測値・事件譚」で、禁止文の正本ではない
//   （層ごとの持ち分は documents/adr/README.md の「正本の分割規約」）。
//   ここで検査するのは、その役割を果たせる形になっているかどうか。
//
//   Related tests の実在検査に Status: Superseded の免除があるのは、
//   **決定を覆す実体が「ガードテストを消すこと」**だから。免除が無いと
//   Superseded にした瞬間に CI が赤くなり、逃げ道が「歴史であるはずの
//   ADR を書き換える」ことしか無くなる。
// ─────────────────────────────────────────────────────────────
const ADR_REQUIRED_HEADINGS = [
  'Context', 'Decision', 'Rejected alternatives',
  'Consequences', 'Evidence', 'Related tests',
]
const ADR_STATUSES = ['Accepted', 'Superseded', 'Deprecated']

function adrFiles() {
  return listFiles('documents/adr',
    (f) => /^\d{4}-.+\.md$/.test(f) && f !== '0000-template.md')
}

// メタ表（本文冒頭の2列テーブル）から1項目を取る。
function adrMeta(text, key) {
  const mt = text.match(new RegExp('^\\|\\s*' + key + '\\s*\\|(.*)\\|\\s*$', 'm'))
  return mt ? mt[1].trim() : null
}

// 「## 見出し」の直後から次の「## 」までを返す。見出しが無ければ null。
function adrSection(text, heading) {
  const mt = text.match(new RegExp('^## ' + heading + '\\s*$', 'm'))
  if (!mt) return null
  const rest = text.slice(mt.index + mt[0].length)
  const next = rest.search(/^## /m)
  return next < 0 ? rest : rest.slice(0, next)
}

function checkADR() {
  const files = adrFiles()
  if (!files.length) return
  const indexText = exists('documents/adr/README.md') ? readText('documents/adr/README.md') : ''
  const seen = new Map()
  const supersedes = new Map()
  const supersededBy = new Map()
  const numsIn = (v) => (v && v !== 'なし') ? [...v.matchAll(/(\d{4})/g)].map((mt) => mt[1]) : []

  for (const f of files) {
    const rel = 'documents/adr/' + f
    const text = readText(rel)
    const num = f.slice(0, 4)

    for (const h of ADR_REQUIRED_HEADINGS) {
      if (adrSection(text, h) === null) err(`ADR に必須見出しが無い: ${rel} → ## ${h}`)
    }

    if (!new RegExp('^# ADR-' + num + ':', 'm').test(text)) {
      err(`ADR の番号がファイル名と本文で食い違う: ${rel} → 本文の見出しが「# ADR-${num}: 」で始まっていない`)
    }
    if (seen.has(num)) err(`ADR 番号の重複: ${num}（${seen.get(num)} と ${rel}）`)
    seen.set(num, rel)

    const status = adrMeta(text, 'Status')
    if (!status) err(`ADR のメタ表に Status が無い: ${rel}`)
    else if (!ADR_STATUSES.includes(status)) {
      err(`ADR の Status が不正: ${rel} → 「${status}」（${ADR_STATUSES.join(' / ')} のいずれか）`)
    }

    const date = adrMeta(text, 'Date')
    if (!/^\d{4}-\d{2}-\d{2}$/.test(date || '')) {
      err(`ADR のメタ表の Date が YYYY-MM-DD でない: ${rel} → 「${date}」`)
    }
    const sources = adrMeta(text, 'Sources')
    if (!sources || sources === 'なし') err(`ADR のメタ表に Sources が無い: ${rel}`)

    // 却下案が1つも無いなら、それは決定ではなく仕様。documents/reverse/ の担当。
    const rejected = adrSection(text, 'Rejected alternatives') || ''
    const rejectedItems = rejected.split(/\r?\n/).filter((l) => /^\s*[-*] \S/.test(l))
    if (!rejectedItems.length) {
      err(`ADR の Rejected alternatives が実質空: ${rel}（却下案が無いなら決定ではなく仕様。documents/reverse/ の担当）`)
    }

    // 実測が無い決定でも空欄にはしない（「実測なし — 理由」を書く）。
    if (!(adrSection(text, 'Evidence') || '').trim()) {
      err(`ADR の Evidence が空: ${rel}（実測が無いなら「実測なし — 理由」と書く）`)
    }

    if (status !== 'Superseded') {
      const related = adrSection(text, 'Related tests') || ''
      for (const mt of related.matchAll(/`(src\/[\w./-]+)`/g)) {
        if (!exists(mt[1])) err(`ADR の Related tests が実在しない: ${rel} → ${mt[1]}`)
      }
    }

    supersededBy.set(num, numsIn(adrMeta(text, 'Superseded-by')))
    supersedes.set(num, numsIn(adrMeta(text, 'Supersedes')))
    if (supersededBy.get(num).length && status !== 'Superseded') {
      err(`ADR に Superseded-by があるのに Status が Superseded でない: ${rel}`)
    }

    if (!indexText.includes(f)) {
      err(`ADR が索引に無い: ${rel}（documents/adr/README.md の索引表に1行足すこと）`)
    }
  }

  // Supersede は相互リンクでなければならない（片方向だと歴史がたどれない）。
  const reciprocal = (from, to, label) => {
    for (const [num, targets] of from) {
      for (const t of targets) {
        if (!seen.has(t)) { err(`ADR-${num} の ${label} が実在しない ADR-${t} を指している`); continue }
        if (!(to.get(t) || []).includes(num)) {
          err(`ADR の Supersede が片方向: ADR-${num} の ${label} は ADR-${t} を指しているが、逆向きが書かれていない`)
        }
      }
    }
  }
  reciprocal(supersededBy, supersedes, 'Superseded-by')
  reciprocal(supersedes, supersededBy, 'Supersedes')
}


// ─────────────────────────────────────────────────────────────
// 4-e. ADR の番号帯（documents/adr/README.md「番号の付け方」の表）
//
//   帯は「壊れたときに同じ場所を読み直すことになる範囲」で切ってある。
//   刻みが10番幅だった頃、4つの帯が満杯になり、中身が MCP の ADR 3本が
//   空いていた「開発規約と資料」帯へ逃げた。番号は採番後不変なので、
//   逃がした瞬間に気づかなければ恒久的にずれたまま残る（経緯は ADR-0805）。
//
//   だから「満杯になってから」ではなく **空きが ADR_BAND_MIN_FREE を切った時点で**
//   落とす。落ちたときの正しい対応は帯を広げるか新しい帯を切ることで、
//   別の帯へ逃がすことではない。
// ─────────────────────────────────────────────────────────────
const ADR_BAND_MIN_FREE = 10

// README の「## 番号の付け方」節から `| NNNN-NNNN | サブシステム |` を拾う。
function adrBands() {
  const rel = 'documents/adr/README.md'
  const text = readText(rel)
  const head = text.match(/^## 番号の付け方\s*$/m)
  if (!head) { err(`${rel} に「## 番号の付け方」節が無い（番号帯の正本）`); return [] }
  const rest = text.slice(head.index + head[0].length)
  const next = rest.search(/^## /m)
  const section = next < 0 ? rest : rest.slice(0, next)
  const bands = []
  for (const mt of section.matchAll(/^\|\s*(\d{4})-(\d{4})\s*\|\s*([^|]+?)\s*\|\s*$/gm)) {
    bands.push({ from: Number(mt[1]), to: Number(mt[2]), name: mt[3] })
  }
  return bands
}

function checkADRBands() {
  const bands = adrBands()
  if (!bands.length) {
    err('documents/adr/README.md の「番号の付け方」に番号帯の表が無い')
    return
  }

  for (const b of bands) {
    if (b.from > b.to) err(`ADR の番号帯の範囲が逆: ${b.from}-${b.to}（${b.name}）`)
  }

  // 帯の重なり。重なると「どちらの帯に属するか」が読み手ごとに変わる。
  const sorted = [...bands].sort((a, b) => a.from - b.from)
  for (let i = 1; i < sorted.length; i++) {
    if (sorted[i].from <= sorted[i - 1].to) {
      err('ADR の番号帯が重なっている: ' +
        `${sorted[i - 1].from}-${sorted[i - 1].to}（${sorted[i - 1].name}）と ` +
        `${sorted[i].from}-${sorted[i].to}（${sorted[i].name}）`)
    }
  }

  const nums = adrFiles().map((f) => Number(f.slice(0, 4)))
  for (const n of nums) {
    if (!bands.some((b) => n >= b.from && n <= b.to)) {
      err(`ADR-${String(n).padStart(4, '0')} がどの番号帯にも入っていない` +
        '（documents/adr/README.md「番号の付け方」の表に帯を足すこと）')
    }
  }

  // 空き枯渇。予約帯（0件）は正常なので、使用数ではなく残り空きだけを見る。
  for (const b of bands) {
    const used = nums.filter((n) => n >= b.from && n <= b.to).length
    const free = (b.to - b.from + 1) - used
    if (free < ADR_BAND_MIN_FREE) {
      err(`ADR の番号帯の空きが足りない: ${b.from}-${b.to}（${b.name}）は残り ${free}。` +
        '帯を広げるか新しい帯を切ること。別の帯へ逃がすと恒久的にずれる（ADR-0805）')
    }
  }
}


// ─────────────────────────────────────────────────────────────
// 8. AI エージェント向け資料（AGENTS.md / CLAUDE.md / .claude/skills）
//    分割の設計と却下案: documents/adr/0804-split-claude-md-into-skills.md
//
//    AGENTS.md と CLAUDE.md は毎セッション全文が読み込まれる入口なので、
//    太るとすべてのタスクの常時コンテキストを食う。上限を機械で固定する。
//    **上限に当たったら上限を上げず、中身を .claude/skills/ へ落とすこと。**
// ─────────────────────────────────────────────────────────────
const SKILLS_DIR = '.claude/skills'
const AGENTS_MD_MAX_BYTES = 24000   // LF 正規化後の実測 約17.5KB + 余裕。上げる前に分割を検討する
const CLAUDE_MD_MAX_LINES = 40

// 作業ツリーは CRLF なので、バイト計測・正規表現の前に必ず LF へ正規化する
const normalizeLF = (s) => s.replace(/\r\n/g, '\n')

function skillNames() {
  return listFiles(SKILLS_DIR, (d) => exists(`${SKILLS_DIR}/${d}/SKILL.md`)).sort()
}

function checkAgentEntrypoints() {
  if (!exists('AGENTS.md')) { err('AGENTS.md が無い（AI エージェント共通の入口）'); return }
  const agents = normalizeLF(readText('AGENTS.md'))
  const bytes = Buffer.byteLength(agents, 'utf8')
  if (bytes > AGENTS_MD_MAX_BYTES) {
    err(`AGENTS.md が ${bytes} バイト（上限 ${AGENTS_MD_MAX_BYTES}）。` +
      '上限を上げるのではなく、領域別の内容を .claude/skills/ へ移してルーティング表に載せること')
  }
  if (!agents.includes('<!-- ROUTING-TABLE:BEGIN')) err('AGENTS.md にルーティング表のマーカーが無い')

  if (!exists('CLAUDE.md')) { err('CLAUDE.md が無い（Claude Code の入口）'); return }
  const claude = normalizeLF(readText('CLAUDE.md'))
  if (!/^@AGENTS\.md\s*$/m.test(claude)) {
    err('CLAUDE.md に `@AGENTS.md` の行が無い（Claude Code に AGENTS.md が読み込まれない）')
  }
  const claudeLines = claude.split('\n').length
  if (claudeLines > CLAUDE_MD_MAX_LINES) {
    err(`CLAUDE.md が ${claudeLines} 行（上限 ${CLAUDE_MD_MAX_LINES}）。` +
      '規約の正本は AGENTS.md と .claude/skills/。CLAUDE.md は入口だけを持つこと')
  }

  // 他AIツールの入口は導線だけを持つ。規約本文の複製は必ずドリフトする
  for (const rel of ['.github/copilot-instructions.md', '.cursor/rules/gkill.mdc']) {
    if (!exists(rel)) { err(`AI 入口ファイルが無い: ${rel}`); continue }
    const pointer = normalizeLF(readText(rel))
    if (!pointer.includes('AGENTS.md')) err(`AI 入口が AGENTS.md を指していない: ${rel}`)
    if (Buffer.byteLength(pointer, 'utf8') > 4096) err(`AI 入口が大きすぎる: ${rel}（導線だけにする）`)
    if (/してはいけない|してはならない/.test(pointer)) {
      err(`AI 入口に規約本文が書かれている: ${rel}（正本は AGENTS.md と .claude/skills/）`)
    }
  }
  if (!exists('.gemini/settings.json')) err('AI 入口ファイルが無い: .gemini/settings.json')
}

function checkSkills() {
  const names = skillNames()
  if (!names.length) {
    err(`規約スキルが1つも見つからない: ${SKILLS_DIR}/*/SKILL.md` +
      '（.gitignore が /.claude/* + !/.claude/skills/ になっているか確認。' +
      'ディレクトリごと無視すると CI の checkout に存在せず、検査が静かにゼロ件になる）')
    return
  }
  const agents = exists('AGENTS.md') ? normalizeLF(readText('AGENTS.md')) : ''
  const tableMatch = agents.match(/<!-- ROUTING-TABLE:BEGIN[\s\S]*?<!-- ROUTING-TABLE:END -->/)
  const table = tableMatch ? tableMatch[0] : ''
  const linked = new Set()
  for (const mt of table.matchAll(/\]\((\.claude\/skills\/[\w-]+\/SKILL\.md)\)/g)) linked.add(mt[1])

  for (const name of names) {
    const rel = `${SKILLS_DIR}/${name}/SKILL.md`
    const text = normalizeLF(readText(rel))
    const fm = text.match(/^---\n([\s\S]*?)\n---\n/)
    if (!fm) { err(`SKILL.md に frontmatter が無い: ${rel}`); continue }
    const nameLine = fm[1].match(/^name:\s*(.+?)\s*$/m)
    if (!nameLine || nameLine[1] !== name) {
      err(`SKILL.md の name がディレクトリ名と違う: ${rel} → 「${nameLine ? nameLine[1] : '(無し)'}」`)
    }
    // description は二重引用符でくくった1物理行（オンデマンド発動の唯一の手がかり）
    const descLine = fm[1].match(/^description:\s*"(.+)"\s*$/m)
    if (!descLine) {
      err(`SKILL.md の description が無いか、二重引用符1行の形式でない: ${rel}`)
    } else {
      const d = descLine[1]
      if (d.length < 80) err(`description が短すぎて発動精度が出ない: ${rel}（${d.length}字）`)
      if (d.length > 1024) err(`description が長すぎる（常時コンテキストを食う）: ${rel}（${d.length}字）`)
      if (!/(src\/|\.claude\/|documents\/|resources\/|package\.json|AGENTS\.md|CLAUDE\.md|\.(ts|go|vue|mjs)\b)/.test(d)) {
        err(`description に発動の手がかり（パスやファイル名）が無い: ${rel}`)
      }
    }
    if (!linked.has(rel)) {
      err(`スキルが AGENTS.md のルーティング表に無い: ${rel}` +
        '（表に行が無いスキルは、パス連動のスキル機構を持たないエージェントから永遠に読まれない）')
    }
    // 1スキル = SKILL.md 1ファイル。補助 .md の散在は「索引に載らず読まれない資料」になる
    for (const f of listFiles(`${SKILLS_DIR}/${name}`, (f2) => f2.endsWith('.md') && f2 !== 'SKILL.md')) {
      err(`スキルに SKILL.md 以外の .md がある: ${SKILLS_DIR}/${name}/${f}（1スキル=1ファイル。内容は SKILL.md へ）`)
    }
  }
  for (const p of linked) {
    const dirName = p.split('/')[2]
    if (!names.includes(dirName)) err(`ルーティング表にあるスキルが実在しない: ${p}`)
  }
}

// ADR の | Sources | が指す資料ファイルと節名の実在。
// 節が別ファイルへ移った・改名されたのに出典が旧位置を指し続ける事故は、
// これまでどの検査にも掛かっていなかった（今回の分割で 20 ADR の出典が動いた）。
// 節名の照合はバッククォートと『』/「」の揺れを吸収した部分文字列一致。
function checkADRSources() {
  const norm = (s) => s.replace(/`/g, '').replace(/『/g, '「').replace(/』/g, '」')
  for (const f of adrFiles()) {
    const rel = 'documents/adr/' + f
    const sources = adrMeta(readText(rel), 'Sources')
    if (!sources) continue
    for (const mt of sources.matchAll(/`([\w./-]+\.md)`\s*(?:の)?「([^」]+)」/g)) {
      const [, file, section] = mt
      if (!exists(file)) { err(`ADR の Sources が実在しないファイルを指している: ${rel} → ${file}`); continue }
      if (!norm(normalizeLF(readText(file))).includes(norm(section))) {
        err(`ADR の Sources の節が見つからない: ${rel} → ${file}「${section}」` +
          '（節の移動・改名。出典のファイルパスを追随させること）')
      }
    }
  }
}

// 個人情報・実環境情報の混入検査（AGENTS.md「AI エージェントへの約束」の機械化）。
// 公開リポジトリなので、実在の利用者ID・端末のローカル絶対パス・メールアドレスを資料に書かない。
// パターンで表せない固有の NG 語（実在の名前など）は、それ自体をコミットすると本末転倒なので、
// gitignore 済みの verify_docs_personal_ngwords.local.txt（1行1語）に置くとその環境でだけ検査に加わる。
// 個人情報検査の対象ファイル列挙（資料 Markdown に加えて src/ のコード・テストと
// resources/manual_src/ の原稿）。2026-08-24 の監査で、資料層だけの検査では
// ソースコメントへの混入（実ハンドルを使った例示など）を原理的に検出できないと分かったため広げた。
// gitignore 済みのビルド生成物（.gradle / build 等）は DOC_FILENAME_SKIP_DIRS で外れる。
// 依存 OSS のライセンス原文（ルート直下の LICENSES_DEPENDENCE）は原著者のメールを含むが、
// 対象パスに入っていないので誤検出しない。
function personalInfoScanFiles() {
  const exts = /\.(go|ts|vue|mjs|js|kt|java|json|html|md|css|ps1|sh|kts|gradle|ya?ml|xml|properties|txt|csv|svg|db)$/
  // 「追跡済み + 未追跡だが ignore されていない」= リポジトリに入り得るファイルだけを見る。
  // ファイルシステム走査だと gitignore 済みのローカル設定（Android の local.properties や
  // ビルド生成物）まで拾って偽陽性になるし、逆に ignore されていない置き忘れは拾いたい。
  const out = execSync('git ls-files -z --cached --others --exclude-standard', { cwd: ROOT })
  return out.toString('utf8').split('\0').filter(Boolean)
    .filter((rel) => exts.test(rel) && /^(src|resources|documents|\.github)\//.test(rel))
    // サンプルデータは実データ由来を許容する運用（2026-08-24 の判断）。検査対象外。
    // 3文字級の短い NG 語は DB 内の base64 や複合語の一部に偶発一致するため、
    // ここを対象に戻すなら語側を前後にアンダースコア等の区切りを付けた形へ寄せること。
    .filter((rel) => !rel.startsWith('resources/gkill_sample_data/'))
}

// ZIP コンテナ文書（xlsx / docx / zip）の列挙。Office 文書は ZIP+deflate なので、
// 生バイトの UTF-8 走査では内部 XML の文字列が原理的に見えない（2026-08-30 の監査で、
// 公開 xlsx の内部 XML に環境固有語が残っていたのに checkPersonalInfo が素通りしていた）。
// そのため ZIP は展開してテキスト系エントリだけを同じ検査に通す。
const PERSONAL_INFO_ZIP_EXEMPT = new Set([
  // 既存の巨大証跡3件は実データ由来を許容する運用で対象外（gkill_sample_data と同じ判断。
  // 数十〜100MB の展開回避も兼ねる）。新規に追加した ZIP/Office 文書は検査対象に入る。
  'documents/evidences/gkill_全体テスト1_エビデンス.zip',
  'documents/evidences/gkill_全体テスト2_エビデンス.xlsx',
  'documents/evidences/gkill_全体テスト2_フィードバック分テスト_エビデンス.xlsx',
])

function personalInfoZipFiles() {
  const out = execSync('git ls-files -z --cached --others --exclude-standard', { cwd: ROOT })
  return out.toString('utf8').split('\0').filter(Boolean)
    .filter((rel) => /\.(xlsx|docx|pptx|zip)$/.test(rel) && /^(src|resources|documents|\.github)\//.test(rel))
    .filter((rel) => !rel.startsWith('resources/gkill_sample_data/'))
    .filter((rel) => !PERSONAL_INFO_ZIP_EXEMPT.has(rel))
}

// ZIP の central directory を直接読み、テキスト系エントリを { name, text } で返す。
// 依存を増やさないための最小実装（deflate は zlib、無圧縮はそのまま）。
// ZIP64・未知の圧縮方式・壊れたヘッダは「読めないので検査できない」を err にする
// （黙って素通りすると、この検査を足した理由がそのまま再発する）。
function readZipTextEntries(rel) {
  const buf = fs.readFileSync(abs(rel))
  // End of Central Directory (0x06054b50) を末尾から探す（ZIP コメントは最大 64KB）
  let eocd = -1
  const scanEnd = Math.max(0, buf.length - 65557)
  for (let i = buf.length - 22; i >= scanEnd; i--) {
    if (buf.readUInt32LE(i) === 0x06054b50) { eocd = i; break }
  }
  if (eocd < 0) {
    err(`ZIP として読めないため個人情報検査ができない: ${rel}`)
    return []
  }
  const count = buf.readUInt16LE(eocd + 10)
  const cdOffset = buf.readUInt32LE(eocd + 16)
  if (count === 0xffff || cdOffset === 0xffffffff) {
    err(`ZIP64 形式は個人情報検査が未対応: ${rel}（検査を拡張するか、理由を書いて対象から外すこと）`)
    return []
  }
  const entries = []
  let p = cdOffset
  for (let i = 0; i < count; i++) {
    if (p + 46 > buf.length || buf.readUInt32LE(p) !== 0x02014b50) {
      err(`ZIP の central directory が壊れているため個人情報検査ができない: ${rel}`)
      break
    }
    const method = buf.readUInt16LE(p + 10)
    const compSize = buf.readUInt32LE(p + 20)
    const nameLen = buf.readUInt16LE(p + 28)
    const extraLen = buf.readUInt16LE(p + 30)
    const commentLen = buf.readUInt16LE(p + 32)
    const localOffset = buf.readUInt32LE(p + 42)
    const name = buf.subarray(p + 46, p + 46 + nameLen).toString('utf8')
    p += 46 + nameLen + extraLen + commentLen
    // 画像等のバイナリは対象外（UTF-8 に無理やり載せると偶発一致の偽陽性だけが増える）
    if (!/\.(xml|rels|txt|csv|json|html)$/i.test(name)) continue
    if (compSize === 0) continue
    const lNameLen = buf.readUInt16LE(localOffset + 26)
    const lExtraLen = buf.readUInt16LE(localOffset + 28)
    const dataStart = localOffset + 30 + lNameLen + lExtraLen
    const data = buf.subarray(dataStart, dataStart + compSize)
    if (method === 0) entries.push({ name, text: data.toString('utf8') })
    else if (method === 8) entries.push({ name, text: zlib.inflateRawSync(data).toString('utf8') })
    else err(`ZIP の圧縮方式 ${method} は個人情報検査が未対応: ${rel} → ${name}`)
  }
  return entries
}

// ─────────────────────────────────────────────────────────────
// 6-e. 改名済みの綴りが復活していないか（ADR-0806）
//
//   ADR-0802 は plaing の綴りを凍結していたが、その決定は「文書だけが防御線で、
//   一括置換で壊せる状態にある」と ADR 自身が書いていたとおり機械検査を持たなかった。
//   ADR-0806 で playing へ全面改名したので、今度は逆向きに固定する。
//   pre-commit が npm run verify_docs を機械強制するので、ここが一番強い止め金になる。
//
//   除外は「過去にそう書かれた事実」を記録するものだけ。増やすときは ADR-0806 の
//   Consequences に理由を1行足すこと（allowlist が育つと防御線がまた文書へ戻る）。
// ─────────────────────────────────────────────────────────────
const RETIRED_SPELLINGS = [
  {
    label: 'plaing',
    correct: 'playing',
    adr: 'ADR-0806',
    re: /plaing/i,
    exemptFiles: new Set([
      // 凍結を決めた ADR そのもの。documents/adr/README.md「ファイル名の slug は改名しない」
      // に従い、ファイル名も本文もそのまま残す
      'documents/adr/0802-freeze-plaing-spelling.md',
      // 改名を決めた ADR。旧綴りを名指しできないと、何を直したのか書けない
      'documents/adr/0806-fix-spellings-instead-of-freezing.md',
      // この検査自身。旧綴りは検出パターンとして書く必要がある
      'src/tools/verify_docs.mjs',
    ]),
    // 公開済みの歴史記録。書き換えると「そのとき何を配ったか」が消える
    exemptPrefixes: ['documents/releasenote/'],
    // 行に含まれていれば見逃す語。
    //   0802-freeze-plaing-… — 上の ADR へのリンク先ファイル名
    //   retired-spelling-ok  — 旧綴りを名指しする必要がある1行だけの明示的な逃げ道。
    //                          用語集の「凍結された綴り」表のように、記録することが目的の行に付ける
    exemptLine: /0802-freeze-plaing-spelling|retired-spelling-ok/,
  },
]

// 中身の綴りを直せない・直す意味がないもの。バイナリは NUL 判定でも落とすが、
// 拡張子で先に弾いて数十MBの読み込みを避ける。
const RETIRED_SPELLING_SKIP_EXT =
  /\.(png|jpe?g|gif|webp|ico|pdf|zip|xlsx|docx|db|jar|so|dll|exe|keystore|jks|woff2?|ttf|mp4|webm)$/i

function checkRetiredSpellings() {
  // 追跡済み + 未追跡だが ignore されていない = リポジトリに入り得るファイルだけを見る
  // （checkPersonalInfo と同じ列挙。生成物やローカル設定で偽陽性を出さないため）。
  const out = execSync('git ls-files -z --cached --others --exclude-standard', { cwd: ROOT })
  const files = out.toString('utf8').split('\0').filter(Boolean)
  for (const spell of RETIRED_SPELLINGS) {
    for (const rel of files) {
      if (spell.exemptFiles.has(rel)) continue
      if (spell.exemptPrefixes.some((p) => rel.startsWith(p))) continue
      if (RETIRED_SPELLING_SKIP_EXT.test(rel)) continue
      let buf
      try { buf = fs.readFileSync(abs(rel)) } catch { continue }
      if (buf.includes(0)) continue
      const text = buf.toString('utf8')
      if (!spell.re.test(text)) continue
      normalizeLF(text).split('\n').forEach((line, i) => {
        if (!spell.re.test(line)) return
        if (spell.exemptLine.test(line)) return
        err(`改名済みの綴りが復活している: ${rel}:${i + 1} → 「${spell.label}」は`
          + `「${spell.correct}」へ改名済み（${spell.adr}）`)
      })
    }
  }
}

function checkPersonalInfo() {
  const patterns = [
    [/[A-Za-z]:\\+Users\\+(?![〈<]|user(?:name)?\b)[A-Za-z0-9]/, 'Windows のユーザープロファイル実パス'],
    [/\/(?:home|Users)\/(?!user\/|〈|<)[a-z0-9_-]{3,}\//, 'ホームディレクトリの実パス'],
    [/[A-Za-z0-9._%+-]+@(?!example\.)[A-Za-z0-9-]+(?:\.[A-Za-z0-9-]+)*\.[A-Za-z]{2,}/, 'メールアドレス'],
  ]
  // NG 語の照合は大文字小文字を無視する（小文字で書いた1行が大文字混じりの出現も拾う）。
  // 1行1語・部分一致。語そのものをコミットしないための gitignore 済みローカルファイル。
  const ngWords = exists('verify_docs_personal_ngwords.local.txt')
    ? normalizeLF(readText('verify_docs_personal_ngwords.local.txt')).split('\n')
      .map((w) => w.trim().toLowerCase()).filter(Boolean)
    : []
  const targets = [...new Set([...docMarkdownFiles(), ...personalInfoScanFiles()])]
  for (const rel of targets) {
    const text = normalizeLF(readText(rel))
    for (const [re, label] of patterns) {
      const mt = text.match(re)
      if (mt) {
        err(`個人情報の疑い（${label}）: ${rel} → 「${mt[0].slice(0, 40)}」` +
          '（$HOME や 〈ユーザー名〉 のプレースホルダに置き換えること）')
      }
    }
    if (ngWords.length !== 0) {
      const lowered = text.toLowerCase()
      for (const w of ngWords) {
        if (lowered.includes(w)) err(`個人情報の疑い（ローカル NG 語）: ${rel} に「${w}」`)
      }
    }
  }
  // ZIP コンテナ文書の内部テキスト。パターン検査は全部適用するが、NG 語は4文字以上に限る:
  // 表計算 XML には数値・座標・base64 断片が大量にあり、2〜3文字の短語は偶発一致だらけで
  // 信号にならない（gkill_sample_data を対象外にしたのと同じ判断）。
  const zipNgWords = ngWords.filter((w) => w.length >= 4)
  for (const rel of personalInfoZipFiles()) {
    for (const entry of readZipTextEntries(rel)) {
      const text = normalizeLF(entry.text)
      const where = `${rel} 内 ${entry.name}`
      for (const [re, label] of patterns) {
        const mt = text.match(re)
        if (mt) {
          err(`個人情報の疑い（${label}）: ${where} → 「${mt[0].slice(0, 40)}」` +
            '（$HOME や 〈ユーザー名〉 のプレースホルダに置き換えること）')
        }
      }
      if (zipNgWords.length !== 0) {
        const lowered = text.toLowerCase()
        for (const w of zipNgWords) {
          if (lowered.includes(w)) err(`個人情報の疑い（ローカル NG 語）: ${where} に「${w}」`)
        }
      }
    }
  }
}


// ソース内アンカーコメント（規約スキルへの参照）の実在検査。
// コメント内の参照は checkLinks に載らないので、スキルの改名・削除で静かに古びる。
// 高リスクファイルの先頭に「編集前に読む: .claude/skills/<name>/SKILL.md」を置く運用（ADR-0804）。
function checkSkillAnchors() {
  const re = /\.claude\/skills\/[\w-]+\/SKILL\.md/g
  const exts = /\.(go|ts|vue|mjs|kt)$/
  const walk = (dir) => {
    for (const e of fs.readdirSync(dir, { withFileTypes: true })) {
      const p = path.join(dir, e.name)
      if (e.isDirectory()) {
        if (DOC_FILENAME_SKIP_DIRS.has(e.name)) continue
        walk(p)
        continue
      }
      if (!exts.test(e.name)) continue
      const text = fs.readFileSync(p, 'utf8')
      for (const mt of text.matchAll(re)) {
        if (!exists(mt[0])) {
          err('ソースのアンカーコメントが指すスキルが実在しない: ' +
            `${path.relative(ROOT, p).split(path.sep).join('/')} → ${mt[0]}`)
        }
      }
    }
  }
  walk(abs('src'))
}

// ─────────────────────────────────────────────────────────────
// メイン
// ─────────────────────────────────────────────────────────────
function main() {
  const m = computeMetrics()

  if (process.argv.includes('--list')) {
    console.log('実測メトリクス:')
    console.log(JSON.stringify(m, null, 2))
    return
  }

  // 構造パリティ（翻訳同期の指針）は明示実行のみ。通常/CIはクリーンに保つ。
  if (process.argv.includes('--parity')) {
    checkManualParity(true)
    return
  }

  checkCounts(m)
  checkLinks()
  checkPaths()
  checkDocFilenames()
  checkADR()
  checkADRBands()
  checkSkills()
  checkAgentEntrypoints()
  checkADRSources()
  checkPersonalInfo()
  checkRetiredSpellings()
  checkSkillAnchors()
  checkMermaid()
  checkManuals()
  checkManualTerminology()
  checkManualHelpPages()

  // 警告とエラーは stderr へ。stdout に出すと、成功時の出力だけを捨てる
  // パイプラインや stdout/stderr を分けている CI で失敗理由が消える
  // （src/tools の他のCLIは元から console.error を使っている）。
  if (warnings.length) {
    console.warn(`\n⚠️  警告 ${warnings.length}件:`)
    for (const w of warnings) console.warn('  - ' + w)
  }
  if (errors.length) {
    console.error(`\n❌ エラー ${errors.length}件:`)
    for (const e of errors) console.error('  - ' + e)
    console.error(`\n実測メトリクス: handlers=${m.handlers} reqRes=${m.reqRes} views=${m.views} dialogs=${m.dialogs} pages=${m.pages} endpoints=${m.endpoints} i18nKeys=${m.i18nKeys} total=${m.componentTotal}`)
    process.exit(1)
  }
  console.log(`✅ docs 検証OK（handlers=${m.handlers} reqRes=${m.reqRes} views=${m.views} dialogs=${m.dialogs} pages=${m.pages} endpoints=${m.endpoints} i18nKeys=${m.i18nKeys} total=${m.componentTotal}${warnings.length ? `, 警告${warnings.length}件` : ''}）`)
}

main()
