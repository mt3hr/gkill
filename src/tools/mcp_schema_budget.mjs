#!/usr/bin/env node
// 編集前に読む: .claude/skills/gkill-mcp/SKILL.md「tools/list のバイト量は予算ファイルで固定する」
//
// MCP の tools/list のバイト量を予算ファイル（src/mcp/tool-schema-budget.json）と突き合わせる。
// 実測と判定は src/mcp/tool-schema-budget.mjs が正本で、テスト
// （src/mcp/__tests__/tool-schema-budget.test.mjs）も同じ関数を使う。
//
//   node src/tools/mcp_schema_budget.mjs            … 実測と予算を表示。予算外なら exit 1
//   node src/tools/mcp_schema_budget.mjs --update   … 実測を予算ファイルへ書く
//   npm run mcp:schema-budget -- --update           … 同上
//
// 依存なし（Node 標準のみ）。

import process from 'node:process'

import {
  TOOL_SCHEMA_BUDGET_PATH,
  compareToolSchemaBudget,
  describeBudgetRow,
  measureToolSchemaBytes,
  readToolSchemaBudget,
  writeToolSchemaBudget,
} from '../mcp/tool-schema-budget.mjs'

const args = process.argv.slice(2)
const update = args.includes('--update')
const unknown = args.filter((arg) => arg !== '--update')
if (unknown.length !== 0) {
  console.error(`[mcp_schema_budget] unknown argument: ${unknown.join(' ')} (only --update is accepted)`)
  process.exit(2)
}

const current = measureToolSchemaBytes()

if (update) {
  writeToolSchemaBudget(current)
  console.log(`[mcp_schema_budget] wrote ${TOOL_SCHEMA_BUDGET_PATH}`)
  for (const [kind, bytes] of Object.entries(current)) {
    console.log(`  ${kind}: ${bytes} bytes`)
  }
  process.exit(0)
}

let budget = null
try {
  budget = readToolSchemaBudget()
} catch (e) {
  console.error(`[mcp_schema_budget] cannot read ${TOOL_SCHEMA_BUDGET_PATH}: ${e.message}`)
}

let failed = false
for (const row of compareToolSchemaBudget(current, budget)) {
  console.log(`  ${describeBudgetRow(row)}`)
  if (row.verdict !== 'ok') failed = true
}
process.exit(failed ? 1 : 0)
