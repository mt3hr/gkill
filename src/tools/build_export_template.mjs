#!/usr/bin/env node
// 編集前に読む: documents/adr/0903-export-shell-is-a-second-vite-build.md
//
// vite.export.config.ts が出した dist_export/ を1枚の自己完結した HTML へ畳み、
// resources/export/template.html へ書く。これがサーバ（export_html_render.go）が
// データを差し込むガワになる。
//
// 畳むもの:
//   - <script type="module" src="/assets/*.js">  → <script type="module"> …中身… </script>
//   - <link rel="stylesheet" href="/assets/*.css"> → <style> …中身… </style>
//   - CSS 内の url(/assets/*)（@mdi/font の woff2 など） → data: URI
//   - <link rel="icon"> など、残った外部参照の <link> は捨てる
//
// 最後に**外部参照が1つでも残っていないか検査し、残っていればビルドを落とす**。
// ここが唯一の機械的な安全網で、落とすと「出力した HTML が通信する」が静かに通る
// （開いた本人には何も起きず、オフラインの相手の手元でだけ画面が壊れる）。
//
// 依存なし（Node 標準のみ）。

import fs from 'node:fs'
import path from 'node:path'
import process from 'node:process'
import { fileURLToPath } from 'node:url'

const ROOT = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..', '..')
const DIST = path.join(ROOT, 'dist_export')
const OUT_DIR = path.join(ROOT, 'resources', 'export')
const OUT_FILE = path.join(OUT_DIR, 'template.html')

/** 拡張子 → data: URI の MIME。ここに無い拡張子が出てきたら落とす（黙って壊れた参照を残さない） */
const ASSET_MIME = {
  '.woff2': 'font/woff2',
  '.woff': 'font/woff',
  '.ttf': 'font/ttf',
  '.eot': 'application/vnd.ms-fontobject',
  '.png': 'image/png',
  '.jpg': 'image/jpeg',
  '.jpeg': 'image/jpeg',
  '.gif': 'image/gif',
  '.webp': 'image/webp',
  '.svg': 'image/svg+xml',
  '.avif': 'image/avif',
}

function fail(message) {
  console.error(`[build_export_template] ${message}`)
  process.exit(1)
}

/** dist_export からの相対パスでアセットを読む。/assets/x.js も assets/x.js も受ける */
function read_asset(url_path) {
  const rel = url_path.replace(/^\/+/, '').split('?')[0].split('#')[0]
  const abs = path.join(DIST, rel)
  if (!abs.startsWith(DIST)) {
    fail(`アセットの参照が dist_export の外を指している: ${url_path}`)
  }
  if (!fs.existsSync(abs)) {
    fail(`アセットが見つからない: ${url_path}`)
  }
  return fs.readFileSync(abs)
}

/**
 * JS を <script> の中へ置ける形にする。
 *
 * `</script` が含まれていると、ブラウザはそこで script を閉じて残りを HTML として
 * 解釈する（minify 済みの JS でも、文字列リテラルの中に現れうる）。
 * 同じ理由で `<!--` も潰す。
 */
function escape_for_script_element(code) {
  return code
    .replace(/<\/script/gi, '<\\/script')
    .replace(/<!--/g, '<\\!--')
}

/** CSS 内の url(/assets/...) を data: URI へ畳む */
function inline_css_urls(css) {
  return css.replace(/url\(\s*(['"]?)([^'")]+)\1\s*\)/g, (whole, _quote, url) => {
    if (/^(data:|https?:|\/\/)/i.test(url)) {
      // data: はそのまま。http(s) は検査で落ちる（ここで消すと見逃しになる）
      return whole
    }
    const ext = path.extname(url.split('?')[0]).toLowerCase()
    const mime = ASSET_MIME[ext]
    if (!mime) {
      fail(`CSS の url() に data: URI へ畳めない拡張子がある: ${url}`)
    }
    const base64 = read_asset(url).toString('base64')
    return `url(data:${mime};base64,${base64})`
  })
}

function main() {
  const entry = path.join(DIST, 'export.html')
  if (!fs.existsSync(entry)) {
    fail(`dist_export/export.html が無い。先に npm run build_export_shell を通すこと`)
  }
  let html = fs.readFileSync(entry, 'utf8')

  // 1) <script src> を畳む
  html = html.replace(
    /<script\b([^>]*?)\bsrc=(["'])([^"']+)\2([^>]*)><\/script>/gi,
    (_whole, before, _quote, src, after) => {
      if (/^(https?:|\/\/)/i.test(src)) {
        // 外部スクリプトは畳めない。検査で落とす
        return _whole
      }
      const attrs = `${before} ${after}`
      const is_module = /\btype=(["'])module\1/i.test(attrs)
      const code = escape_for_script_element(read_asset(src).toString('utf8'))
      return `<script${is_module ? ' type="module"' : ''}>${code}</script>`
    },
  )

  // 2) <link rel="stylesheet"> を畳む
  html = html.replace(
    /<link\b[^>]*\brel=(["'])stylesheet\1[^>]*>/gi,
    (whole) => {
      const matched = /\bhref=(["'])([^"']+)\1/i.exec(whole)
      if (!matched) {
        return whole
      }
      const href = matched[2]
      if (/^(https?:|\/\/)/i.test(href)) {
        return whole
      }
      const css = inline_css_urls(read_asset(href).toString('utf8'))
      return `<style>${css}</style>`
    },
  )

  // 3) 残った <link>（favicon / modulepreload など）は捨てる。
  //    出力 HTML はアイコンを持たないが、外部参照を残すほうが悪い
  html = html.replace(/<link\b[^>]*>/gi, (whole) => {
    const matched = /\bhref=(["'])([^"']+)\1/i.exec(whole)
    if (!matched) {
      return ''
    }
    if (/^data:/i.test(matched[2])) {
      return whole
    }
    return ''
  })

  // 4) 自己完結の検査。ここが落ちたら、残った参照を畳むか捨てるかを決めること。
  //
  //    **HTML の骨格と、畳んだ中身で見る規則を分ける。** 畳んだ JS には
  //    href="`+e+`" のような文字列テンプレートが普通に入っているので、
  //    HTML 用の正規表現をそのまま当てると必ず誤検知する。
  const violations = []
  const check = (target, re, description) => {
    const found = target.match(re)
    if (found) {
      violations.push(`${description}: ${[...new Set(found)].slice(0, 5).join(' / ')}`)
    }
  }

  // 畳んだ中身を取り出し、骨格側はプレースホルダへ置き換える
  const inlined_bodies = []
  const skeleton = html.replace(
    /<(script|style)\b[^>]*>([\s\S]*?)<\/\1>/gi,
    (_whole, tag, body) => {
      inlined_bodies.push({ tag: tag.toLowerCase(), body })
      return `<${tag}>__GKILL_INLINED__</${tag}>`
    },
  )

  // 骨格: 外部参照の属性が1つも無いこと（data: は許す）
  check(skeleton, /\b(?:src|href)=(["'])(?!data:)[^"']*\1/gi, '外部参照の属性が残っている')

  for (const { tag, body } of inlined_bodies) {
    if (tag === 'style') {
      // CSS: 相対・絶対の url() が残っていないこと
      check(body, /url\(\s*['"]?(?!data:)[./][^'")]*\)/gi, 'CSS の url() に参照が残っている')
      continue
    }
    // JS: 別チャンクを読みに行く形が残っていないこと。
    // /api/... のような文字列は gkill-api.ts がパス表を持っているので必ず現れる。
    // 「呼ばれないこと」は静的には言えないので、通信ゼロは E2E
    // （全通信を abort して開く）が見張る
    check(body, /\bfrom\s*["']\/assets\//g, 'JS に /assets/ への import が残っている')
    check(body, /\bimport\s*\(\s*["']\/[^"']*["']\s*\)/g, 'JS に絶対パスの動的 import が残っている')
    check(body, /__VITE_PRELOAD__|__vitePreload/g, 'JS にチャンクのプリロードが残っている')
    // Service Worker の**登録**だけを見る。navigator.serviceWorker.ready は
    // Web Push の購読コード（設定画面の経路）が持っており、出力 HTML では呼ばれない
    check(body, /serviceWorker\s*\.\s*register|virtual:pwa-register|workbox/g,
      'Service Worker の登録が残っている')
  }

  if (violations.length !== 0) {
    fail(`出力 HTML が自己完結していない:\n  - ${violations.join('\n  - ')}`)
  }
  if (!html.includes('id="gkill-export-data"')) {
    fail('データの差し込み先 id="gkill-export-data" が消えている（export.html を確認すること）')
  }
  if (!html.includes('{{GKILL_EXPORT_TITLE}}')) {
    fail('タイトルの差し込み先 {{GKILL_EXPORT_TITLE}} が消えている')
  }

  fs.mkdirSync(OUT_DIR, { recursive: true })
  fs.writeFileSync(OUT_FILE, html, 'utf8')

  const size_kib = Math.round(Buffer.byteLength(html, 'utf8') / 1024)
  console.log(`[build_export_template] できました: ${path.relative(ROOT, OUT_FILE)} (${size_kib} KiB)`)
}

main()
