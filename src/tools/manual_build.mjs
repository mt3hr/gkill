// マニュアル生成の共有ロジック。
// build_manuals.mjs（書き出し）と verify_docs.mjs（生成鮮度の検証）が共用する。

import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'
import { applyA11y } from './manual_a11y.mjs'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
export const ROOT = path.resolve(__dirname, '..', '..')
export const SRC_DIR = path.join(ROOT, 'resources', 'manual_src')
export const OUT_DIR = path.join(ROOT, 'resources', 'manual')

function fillTemplate(layout, { lang, title, content }) {
  return layout
    .split('{{lang}}').join(lang)
    .split('{{title}}').join(title)
    .split('{{content}}').join(content)
}

function parseFragment(frag, file) {
  const nl = frag.indexOf('\n')
  if (nl < 0) throw new Error(`不正なフラグメント（改行なし）: ${file}`)
  const titleLine = frag.slice(0, nl)
  const m = titleLine.match(/^<!-- title: ([\s\S]*?) -->$/)
  if (!m) throw new Error(`title コメントが見つからない: ${file}`)
  return { title: m[1], content: frag.slice(nl + 1) }
}

// 全ページをメモリ上で生成し、[{ relPath, outPath, content }] を返す（書き出しはしない）。
const readLF = (p) => fs.readFileSync(p, 'utf8').replace(/\r\n/g, '\n')

export function renderAll() {
  const layout = readLF(path.join(SRC_DIR, '_layout.html'))
  const langs = fs.readdirSync(SRC_DIR).filter((d) =>
    fs.statSync(path.join(SRC_DIR, d)).isDirectory())

  const results = []
  for (const lang of langs) {
    const langDir = path.join(SRC_DIR, lang)
    const pages = fs.readdirSync(langDir).filter((f) => f.endsWith('.html'))
    for (const page of pages) {
      const frag = readLF(path.join(langDir, page))
      const { title, content } = parseFragment(frag, `${lang}/${page}`)
      const html = fillTemplate(layout, { lang, title, content: applyA11y(content) })
      results.push({
        relPath: `resources/manual/${lang}/${page}`,
        outPath: path.join(OUT_DIR, lang, page),
        content: html,
      })
    }
  }
  return results
}
