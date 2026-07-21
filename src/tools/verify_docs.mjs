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
    componentTotal: views + dialogs + pages,
    localeKeyCounts,
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
function checkLinks() {
  const dir = 'documents/reverse'
  const mds = listFiles(dir, (f) => f.endsWith('.md'))
  const linkRe = /\]\(([^)]+)\)/g
  for (const f of mds) {
    const rel = path.join(dir, f)
    const text = readText(rel)
    let mt
    while ((mt = linkRe.exec(text)) !== null) {
      let target = mt[1].trim()
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
