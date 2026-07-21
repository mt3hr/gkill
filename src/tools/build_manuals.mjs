#!/usr/bin/env node
// マニュアル生成（resources/manual_src → resources/manual）
//
// 共有レイアウト resources/manual_src/_layout.html に各言語×ページのフラグメントを
// 流し込み、resources/manual/{lang}/{page}.html を生成する。
// 生成ロジックは manual_build.mjs に集約（verify_docs.mjs と共用）。

import fs from 'node:fs'
import path from 'node:path'
import { renderAll } from './manual_build.mjs'

function main() {
  const pages = renderAll()
  for (const { outPath, content } of pages) {
    fs.mkdirSync(path.dirname(outPath), { recursive: true })
    fs.writeFileSync(outPath, content, 'utf8')
  }
  console.log(`生成完了: ${pages.length} ページ → resources/manual/`)
}

main()
