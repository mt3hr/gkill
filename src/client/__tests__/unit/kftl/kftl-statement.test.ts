import { describe, test, expect } from 'vitest'
import '../../helpers/setup-i18n'
import { KFTLStatement } from '@/classes/kftl/kftl-statement'
import type { KFTLStatementLine } from '@/classes/kftl/kftl-statement-line'
import { KFTLStatementLineContext } from '@/classes/kftl/kftl-statement-line-context'
import { KFTLStartMiStatementLine } from '@/classes/kftl/kftl_mi/kftl-start-mi-statement-line'
import { KFTLMiTitleStatementLine } from '@/classes/kftl/kftl_mi/kftl-mi-title-statement-line'
import { KFTLMiBoardNameStatementLine } from '@/classes/kftl/kftl_mi/kftl-mi-board-name-statement-line'
import { KFTLMiEstimateStartTimeStatementLine } from '@/classes/kftl/kftl_mi/kftl-mi-estimate-start-time-statement-line'
import { KFTLMiEstimateEndTimeStatementLine } from '@/classes/kftl/kftl_mi/kftl-mi-estimate-end-time-statement-line'
import { KFTLMiLimitTimeStatementLine } from '@/classes/kftl/kftl_mi/kftl-mi-limit-time-statement-line'

/**
 * 行のつながりは各クラスのコンストラクタに散っていて、どこにも一覧が無い。
 * KFTLStatement.generate_kftl_lines と同じ手順で1行ずつ組み立てて、
 * 次の行のコンストラクタを辿った結果が期待通りの並びかを見る。
 */
function build_statement_lines(line_texts: Array<string>): Array<KFTLStatementLine> {
  const tx_id = 'test_tx'
  const lines = new Array<KFTLStatementLine>()
  let prev_context: KFTLStatementLineContext | null = null
  for (let i = 0; i < line_texts.length; i++) {
    const line_text = line_texts[i]
    const next_line_text = i < line_texts.length - 1 ? line_texts[i + 1] : ''
    const target_id = prev_context?.get_next_statement_line_target_id() ?? 'test_target'
    const is_prototype = prev_context ? prev_context.is_next_prototype() : true
    const context = new KFTLStatementLineContext(tx_id, line_text, target_id, next_line_text, lines.slice(0, i), is_prototype)
    const next_constructor = prev_context?.get_next_statement_line_constructor() ?? null
    lines.push(next_constructor ? next_constructor(line_text, context) : new KFTLStartMiStatementLine(line_text, context))
    prev_context = context
  }
  return lines
}

describe('KFTLStatement', () => {
  test('can be instantiated with text', () => {
    const stmt = new KFTLStatement('テストメモ')
    expect(stmt.get_statement_text()).toBe('テストメモ')
  })

  test('empty text creates valid statement', () => {
    const stmt = new KFTLStatement('')
    expect(stmt.get_statement_text()).toBe('')
  })

  test('lookahead_line_count is defined', () => {
    expect(KFTLStatement.lookahead_line_count).toBeGreaterThan(0)
  })

  // Mi の入力順は AddMi 画面(add-mi-view.vue)と揃える約束になっている。
  // Go 側(kftl_mi.go)も同じ並びなので、崩したら両方直すこと
  test('Mi の行順が AddMi と同じ タイトル→板名→開始→終了→期限 になっている', () => {
    const lines = build_statement_lines(['ーみ', 'テストタスク', '仕事', '2025-03-20', '2025-03-21', '2025-03-22'])
    expect(lines[0]).toBeInstanceOf(KFTLStartMiStatementLine)
    expect(lines[1]).toBeInstanceOf(KFTLMiTitleStatementLine)
    expect(lines[2]).toBeInstanceOf(KFTLMiBoardNameStatementLine)
    expect(lines[3]).toBeInstanceOf(KFTLMiEstimateStartTimeStatementLine)
    expect(lines[4]).toBeInstanceOf(KFTLMiEstimateEndTimeStatementLine)
    expect(lines[5]).toBeInstanceOf(KFTLMiLimitTimeStatementLine)
  })

  test('期限の次の行で Mi の入力は終わる', () => {
    const lines = build_statement_lines(['ーみ', 'テストタスク', '仕事', '2025-03-20', '2025-03-21', '2025-03-22', 'ただのメモ'])
    expect(lines[6]).not.toBeInstanceOf(KFTLMiTitleStatementLine)
    expect(lines[6]).not.toBeInstanceOf(KFTLMiBoardNameStatementLine)
    expect(lines[6]).not.toBeInstanceOf(KFTLMiEstimateStartTimeStatementLine)
    expect(lines[6]).not.toBeInstanceOf(KFTLMiEstimateEndTimeStatementLine)
    expect(lines[6]).not.toBeInstanceOf(KFTLMiLimitTimeStatementLine)
  })
})
