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

import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
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

  // 登録エンドポイント数 = serve.go の HandleFunc 登録数
  const serve = readText('src/server/gkill/api/gkill_server_api/serve.go')
  const endpoints = (serve.match(/HandleFunc\(/g) || []).length

  // アドレス定義数 = gkill_server_api_address.go の XxxMethod 定数。
  //   登録数（endpoints）より多い＝ハンドラ未実装のアドレスがある、という関係。
  //   両方を別々に検査しないと「定義90・登録88」のような古い組が資料に残り続ける。
  const addressGo = readText('src/server/gkill/api/gkill_server_api/gkill_server_api_address.go')
  const endpointMethods = addressGo.match(/Method\s*=\s*"(GET|POST|PUT|DELETE)"/g) || []
  const endpointsDefined = endpointMethods.length
  const endpointsPost = endpointMethods.filter((m) => m.includes('"POST"')).length
  const endpointsGet = endpointMethods.filter((m) => m.includes('"GET"')).length

  // 認証ラッパー別の登録数（serve.go）。PathPrefix 経由の2件は HandleFunc ではないので除く。
  const wrapCount = (name) =>
    (serve.match(new RegExp(`HandleFunc\\([^\\n]*g\\.${name}\\(`, 'g')) || []).length
  const wrapNoAuth = wrapCount('wrapNoAuth')
  const wrapAuth = wrapCount('wrapAuth')
  const wrapAuthRepos = wrapCount('wrapAuthRepos')

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
    endpointsDefined, endpointsPost, endpointsGet,
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
    repsGo: countIn('src/server/gkill/dao/reps', '.go'),
    repsImpl: countIn('src/server/gkill/dao/reps', '.go', { excludeTest: true }),
    usecaseGo: countIn('src/server/gkill/usecase', '.go'),
    usecaseFuncs: countMatches(
      listFilesRec('src/server/gkill/usecase', (f) => f.endsWith('.go')), /^func /gm),
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
  const toolNames = (rel) => {
    if (!exists(rel)) return 0
    return new Set([...readText(rel).matchAll(/name: *"(gkill_[a-z_0-9]+)"/g)].map((m) => m[1])).size
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

  // dao/reps 直下のテストファイル数 / クライアント datas テストファイル数 / Wear OS Kotlin ファイル数
  const repsTestFiles = listFiles('src/server/gkill/dao/reps', (f) => f.endsWith('_test.go')).length
  const datasTestFiles = listFiles('src/client/__tests__/unit/datas', (f) => f.endsWith('.test.ts')).length
  const wearCompanionKt = listFilesRec('src/wear_os/phone_companion/src/main', (f) => f.endsWith('.kt')).length
  const wearWatchKt = listFilesRec('src/wear_os/watch_app/src/main', (f) => f.endsWith('.kt')).length

  return {
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
    datasTestFiles,
    wearCompanionKt,
    wearWatchKt,
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
  add('CLAUDE.md', `${m.dialogs} dialog components`)

  // i18n keys
  add('documents/reverse/folder-structure.md', `${m.i18nKeys}キー/言語`)
  add('CLAUDE.md', `${m.i18nKeys} keys per locale`)

  // endpoints（登録数）
  add('documents/reverse/glossary.md', `${m.endpoints}登録`)
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
    `| Wear OS (\`wear_os/\`) | ${m.wearCompanionTests + m.wearWatchTests} | 9 |`,
  ]
  for (const row of testRows) add('src/ABOUT_TEST.md', row)
  add('src/ABOUT_TEST.md', `Go テスト ${m.pluginGoTests}件`)
  add('documents/reverse/testing-guide.md', `| Go バックエンド | ${m.goTests} | ${m.goTestFiles} |`)
  add('documents/reverse/testing-guide.md', `| フロントエンド ユニット | ${m.unitTests} | ${m.unitTestFiles} |`)
  add('documents/reverse/testing-guide.md', `| フロントエンド E2E | ${m.e2eTests} |`)
  add('documents/reverse/testing-guide.md', `| MCP サーバ | ${m.mcpTests} | ${m.mcpTestFiles} |`)
  add('documents/reverse/testing-guide.md', `| Wear OS | ${m.wearCompanionTests + m.wearWatchTests} |`)
  add('src/server/ABOUT_TEST.md', `${m.goTests}テスト関数、${m.goTestFiles}テストファイル、${m.goTestPkgs}パッケージ`)
  add('src/server/gkill/plugin/sdk/ABOUT_TEST.md', `**${m.sdkTests}テスト（${m.sdkTestFiles}ファイル）**`)
  add('src/mcp/ABOUT_TEST.md', `${m.mcpTests}テスト（${m.mcpTestFiles}ファイル）`)
  add('src/wear_os/ABOUT_TEST.md', `合計${m.wearCompanionTests + m.wearWatchTests}テスト`)

  // ── MCP ツール数（プラグイン2ツールは3サーバ共通）
  const mcpRead = m.mcpReadTools + m.mcpPluginTools
  const mcpWrite = m.mcpWriteTools + m.mcpPluginTools
  const mcpRW = m.mcpReadWriteTools + m.mcpPluginTools
  add('CLAUDE.md', `| Read | ${mcpRead} (`)
  add('CLAUDE.md', `| Write | ${mcpWrite} (`)
  add('CLAUDE.md', `| ReadWrite | ${mcpRW} (`)
  add('resources/manual_src/ja/mcp.html', `<td>gkill-read-server.mjs</td><td>${mcpRead}</td>`)
  add('resources/manual_src/ja/mcp.html', `<td>gkill-write-server.mjs</td><td>${mcpWrite}</td>`)
  add('resources/manual_src/ja/mcp.html', `<td>gkill-readwrite-server.mjs</td><td>${mcpRW}</td>`)

  // ── KFTL ステートメント型数 / glossary 用語数
  add('CLAUDE.md', `KFTL parser (${m.kftlStatementTs} statement types; the Go side has ${m.kftlStatementGo})`)
  add('documents/reverse/folder-structure.md', `KFTLパーサー（${m.kftlStatementTs}ステートメント型）`)
  add('documents/reverse/folder-structure.md', `バックエンド側、${m.kftlStatementGo}ステートメント型`)
  add('CLAUDE.md', `glossary.md (${m.glossaryTerms} terms)`)
  add('documents/reverse/README.md', `ドメイン用語の定義（${m.glossaryTerms}項目）`)
  add('documents/reverse/folder-structure.md', `用語集（${m.glossaryTerms}項目）`)

  // ── docコメント網羅率。
  //    reps のインターフェース契約と HandleXxx は 100% 維持する方針なので、
  //    ここが落ちたら「docを書かずにメソッド/ハンドラを足した」ことを意味する。
  add('CLAUDE.md', `${m.repsIfaceDocumented}/${m.repsIfaceMethods} documented`)
  add('CLAUDE.md', `${m.handlerDocumented}/${m.handlerMethods} handlers documented`)

  // ── ユースケース数（ユニークUC-ID） / シーケンス図数（mermaidブロック）
  add('documents/reverse/usecase.md', `**${m.ucIds}件（ユニークな UC-ID 数）**`)
  add('documents/reverse/README.md', `ユースケース一覧（${m.ucIds}件）`)
  add('documents/reverse/scenario.md', `（${m.ucIds}件、1操作ずつ静的に列挙）`)
  add('documents/reverse/folder-structure.md', `ユースケース一覧（${m.ucIds}件）`)
  add('CLAUDE.md', `usecase.md (${m.ucIds} use cases)`)
  add('documents/reverse/README.md', `シーケンス図（${m.seqDiagrams}本:`)
  add('documents/reverse/folder-structure.md', `シーケンス図（${m.seqDiagrams}本:`)
  add('CLAUDE.md', `sequence-diagrams.md (${m.seqDiagrams} diagrams)`)

  // ── handle_*.go ファイル数（CLAUDE.md / サーバ系README）
  const handlerTests = m.handlers - m.handlersImpl
  add('CLAUDE.md', `HTTP API handlers (${m.handlers} files incl. tests, 1 handler per file)`)
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
  add('CLAUDE.md', `declares \`go ${m.goVersion}\``)

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

  // gkill-api.ts の行数（100の位で丸め。「3,400」表記）
  const apiLines = m.gkillApiLinesApprox.toLocaleString('en-US')
  add('documents/reverse/frontend-architecture.md', `(~${apiLines}行)`)
  add('documents/reverse/frontend-architecture.md', `約${apiLines}行`)
  add('documents/reverse/folder-structure.md', `~${apiLines}行`)
  add('documents/reverse/glossary.md', `約${apiLines}行`)
  add('documents/reverse/program-spec.md', `約${apiLines}行`)
  add('documents/reverse/class-diagrams.md', `約${apiLines}行`)
  add('CLAUDE.md', `(~${apiLines} lines)`)

  // req_res（Go / TypeScript）
  add('documents/reverse/api-endpoints.md', `req_res/\`（${m.reqRes}ファイル）`)
  add('documents/reverse/api-endpoints.md', `構造体（${m.reqRes}ファイル）`)
  add('CLAUDE.md', `Request/response structs for every endpoint (${m.reqRes} files)`)
  add('documents/reverse/folder-structure.md', `リクエスト/レスポンス型（${m.classesApiReqRes}ファイル）`)
  add('documents/reverse/frontend-architecture.md',
    `(${m.classesApiReqRes}ファイル、サーバー側は${m.reqRes}ファイル)`)
  add('documents/reverse/glossary.md', `TypeScript 版入出力型（${m.classesApiReqRes}ファイル）`)

  // エンドポイント: 定義数と登録数は別物なので両方検査する
  add('documents/reverse/api-endpoints.md',
    `${m.endpointsDefined}件定義（${m.endpointsPost} POST + ${m.endpointsGet} GET。うち${m.endpoints}件はハンドラ登録済み`)
  add('documents/reverse/README.md', `全${m.endpointsDefined}エンドポイント（登録済み${m.endpoints}）`)
  add('documents/reverse/README.md', `（${m.endpointsDefined}件定義・${m.endpoints}件登録）`)
  add('documents/reverse/folder-structure.md', `（${m.endpointsDefined}件定義・${m.endpoints}件登録）`)
  add('documents/reverse/folder-structure.md', `${m.endpointsDefined}エンドポイント定義・${m.endpoints}登録`)
  add('documents/reverse/glossary.md', `（${m.endpointsDefined}定義・${m.endpoints}登録）`)
  add('documents/reverse/glossary.md', `全${m.endpointsDefined}エンドポイントのパス・メソッド定義`)
  add('documents/reverse/program-spec.md', `アドレス定義は${m.endpointsDefined}件`)
  add('documents/reverse/program-spec.md',
    `アドレス定義${m.endpointsDefined}件 = ${m.endpointsPost} POST + ${m.endpointsGet} GET`)
  add('documents/reverse/program-spec.md', `（${m.endpointsDefined}件、うち${m.endpoints}件が登録済み）`)
  add('src/server/gkill/api/README.md',
    `## 全エンドポイント一覧（${m.endpointsDefined}エンドポイント定義・${m.endpoints}登録）`)
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
// 検査対象の Markdown（リポジトリルートからの相対パス）。
//   documents/reverse/*.md に加え、README.md 群と ABOUT_TEST.md 群も対象にする。
//   src/README.md の「各サブディレクトリの README / ABOUT_TEST」表がリンク切れのまま
//   放置される事故を防ぐのが主目的。
function docMarkdownFiles() {
  const out = []
  for (const f of listFiles('documents/reverse', (f) => f.endsWith('.md'))) {
    out.push('documents/reverse/' + f)
  }
  if (exists('README.md')) out.push('README.md')
  if (exists('CLAUDE.md')) out.push('CLAUDE.md')
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
  const mdFiles = []
  for (const f of listFiles('documents/reverse', (f) => f.endsWith('.md'))) {
    mdFiles.push(path.join('documents/reverse', f))
  }
  if (exists('CLAUDE.md')) mdFiles.push('CLAUDE.md')

  const codeRe = /`([^`]+)`/g
  const seen = new Set()
  for (const rel of mdFiles) {
    const text = readText(rel)
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
// 6-b. マニュアルの用語検査
//   方針: マニュアルは開発コード名を出さず、UI（i18n）に出てくる語だけを使う。
//   ここでは「UI ラベルとして使われていない開発コード名」を明示的に列挙して禁止する。
//   ロケール値に出るかどうかで自動判定はしない —— NOT_FOUND_*_ERROR_MESSAGE が
//   Kmemo / URLog / ReKyou 等をそのまま出しており、それを根拠にすると素通りしてしまうため。
// ─────────────────────────────────────────────────────────────
const MANUAL_FORBIDDEN_TERMS = [
  'IDFKyou', 'IDF', 'WAN', 'MiReKyou', 'ReKyou', 'Kyou', 'KFTL',
  'Rykv', 'Mkfl', 'Dnote', 'Ryuu', 'Plaing', 'Lantana', 'Nlog',
  'URLog', 'TimeIs', 'Kmemo', 'DVNF', 'RepType',
]
// 例外: Saihate は ja 以外の SAIHATE_APP_NAME がそのまま "Saihate" なので UI ラベル。
// `<code>` の中（server-config.html の rep type 一覧など、UI が生値を表示する箇所）は対象外。

function checkManualTerminology() {
  if (!fs.existsSync(SRC_DIR)) return
  const langs = fs.readdirSync(SRC_DIR).filter((d) =>
    fs.statSync(path.join(SRC_DIR, d)).isDirectory())
  for (const lang of langs) {
    const langDir = path.join(SRC_DIR, lang)
    for (const page of fs.readdirSync(langDir).filter((f) => f.endsWith('.html'))) {
      const body = fs.readFileSync(path.join(langDir, page), 'utf8')
        .replace(/<code>[\s\S]*?<\/code>/g, '')
        .replace(/<!--[\s\S]*?-->/g, '')
      for (const term of MANUAL_FORBIDDEN_TERMS) {
        if (new RegExp(`(^|[^A-Za-z])${term}([^A-Za-z]|$)`).test(body)) {
          err(`マニュアル用語NG: resources/manual_src/${lang}/${page} に開発コード名「${term}」`
            + '（i18n の UI 用語に置き換えること）')
        }
      }
    }
  }
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
  checkMermaid()
  checkManuals()
  checkManualTerminology()
  checkManualHelpPages()

  if (warnings.length) {
    console.log(`\n⚠️  警告 ${warnings.length}件:`)
    for (const w of warnings) console.log('  - ' + w)
  }
  if (errors.length) {
    console.log(`\n❌ エラー ${errors.length}件:`)
    for (const e of errors) console.log('  - ' + e)
    console.log(`\n実測メトリクス: handlers=${m.handlers} reqRes=${m.reqRes} views=${m.views} dialogs=${m.dialogs} pages=${m.pages} endpoints=${m.endpoints} i18nKeys=${m.i18nKeys} total=${m.componentTotal}`)
    process.exit(1)
  }
  console.log(`✅ docs 検証OK（handlers=${m.handlers} reqRes=${m.reqRes} views=${m.views} dialogs=${m.dialogs} pages=${m.pages} endpoints=${m.endpoints} i18nKeys=${m.i18nKeys} total=${m.componentTotal}${warnings.length ? `, 警告${warnings.length}件` : ''}）`)
}

main()
