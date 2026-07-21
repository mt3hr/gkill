// マニュアル本文へのアクセシビリティ変換。
//   - 全 <th> に scope="col" を付与（マニュアルの表はすべて列ヘッダ型・素の <th>）。
//   - 各 <table> の直前に、直近の見出し（h1/h2/h3）テキストから
//     視覚的に隠した <caption class="sr-only"> を挿入。
// <main> ランドマークは _layout.html 側で付与する。
//
// マニュアルの本文は構造が一様（1行1タグ・素の <table>/<th>）なため、
// 行単位の走査で安全に変換できる。

const HEADING_RE = /<h[123][^>]*>([\s\S]*?)<\/h[123]>/i
const stripTags = (s) => s.replace(/<[^>]+>/g, '').trim()

export function applyA11y(content) {
  const lines = content.split('\n')
  const out = []
  let lastHeading = ''

  for (const line of lines) {
    const hm = line.match(HEADING_RE)
    if (hm) lastHeading = stripTags(hm[1])

    // <th> → <th scope="col">（素の <th> のみ。既に scope 付きは触らない）
    const transformed = line.replace(/<th>/g, '<th scope="col">')
    out.push(transformed)

    // <table> 行の直後にキャプションを挿入
    if (/<table>/.test(line)) {
      const indent = (line.match(/^\s*/) || [''])[0]
      const caption = lastHeading || '表'
      out.push(`${indent}  <caption class="sr-only">${caption}</caption>`)
    }
  }
  return out.join('\n')
}
