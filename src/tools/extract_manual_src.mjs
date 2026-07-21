#!/usr/bin/env node
// マニュアル原稿の抽出（一度きりの移行ツール）
//
// 既存 resources/manual/{lang}/{page}.html から <title> と本文（<body>〜<script> の間）を
// バイト単位で抽出し、resources/manual_src/{lang}/{page}.html にフラグメントとして書き出す。
// フラグメント形式: 先頭行 `<!-- title: ... -->`、以降が本文。
//
// 本文の抽出は文字列スライスで行い、元のバイト列を完全に保持する（改行・空行含む）。

import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const ROOT = path.resolve(__dirname, '..', '..')
const SRC_DIR = path.join(ROOT, 'resources', 'manual')
const OUT_DIR = path.join(ROOT, 'resources', 'manual_src')

const BODY_MARKER = '\n<body>\n'
const SCRIPT_MARKER = '\n<script>'

function extractOne(html, file) {
  // 安全ガード: build_manuals 生成済み（<main> を含む）ファイルからの再抽出を禁止。
  // extract は「元の手書きマニュアル」に対する一度きりの移行専用。
  if (html.includes('<main>')) {
    throw new Error(`生成済みファイルからの再抽出は不可（<main> を検出）: ${file}\n` +
      `extract は元の手書きマニュアルに対してのみ実行してください。`)
  }
  const titleM = html.match(/<title>([\s\S]*?)<\/title>/)
  if (!titleM) throw new Error(`<title> が見つからない: ${file}`)
  const title = titleM[1]

  const bi = html.indexOf(BODY_MARKER)
  if (bi < 0) throw new Error(`<body> マーカーが見つからない: ${file}`)
  const si = html.indexOf(SCRIPT_MARKER, bi)
  if (si < 0) throw new Error(`<script> マーカーが見つからない: ${file}`)

  const content = html.slice(bi + BODY_MARKER.length, si) // <body>\n の後〜\n<script> の前（バイト厳密）
  return { title, content }
}

function main() {
  const langs = fs.readdirSync(SRC_DIR).filter((d) =>
    fs.statSync(path.join(SRC_DIR, d)).isDirectory())
  let count = 0
  for (const lang of langs) {
    const langDir = path.join(SRC_DIR, lang)
    const pages = fs.readdirSync(langDir).filter((f) => f.endsWith('.html'))
    const outLangDir = path.join(OUT_DIR, lang)
    fs.mkdirSync(outLangDir, { recursive: true })
    for (const page of pages) {
      // 改行は LF に正規化（Windows の autocrlf 由来の CRLF を吸収）
      const html = fs.readFileSync(path.join(langDir, page), 'utf8').replace(/\r\n/g, '\n')
      const { title, content } = extractOne(html, `${lang}/${page}`)
      const fragment = `<!-- title: ${title} -->\n${content}`
      fs.writeFileSync(path.join(outLangDir, page), fragment, 'utf8')
      count++
    }
  }
  console.log(`抽出完了: ${count} フラグメント → resources/manual_src/`)
}

main()
