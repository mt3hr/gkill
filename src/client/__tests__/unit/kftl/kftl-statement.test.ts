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
import { KFTLStartMiReKyouStatementLine } from '@/classes/kftl/kftl_mirekyou/kftl-start-mi-re-kyou-statement-line'
import { KFTLMiReKyouBoardNameStatementLine } from '@/classes/kftl/kftl_mirekyou/kftl-mi-re-kyou-board-name-statement-line'
import { KFTLMiReKyouEstimateStartTimeStatementLine } from '@/classes/kftl/kftl_mirekyou/kftl-mi-re-kyou-estimate-start-time-statement-line'
import { KFTLMiReKyouEstimateEndTimeStatementLine } from '@/classes/kftl/kftl_mirekyou/kftl-mi-re-kyou-estimate-end-time-statement-line'
import { KFTLMiReKyouLimitTimeStatementLine } from '@/classes/kftl/kftl_mirekyou/kftl-mi-re-kyou-limit-time-statement-line'
import { KFTLMiReKyouTagStatementLine } from '@/classes/kftl/kftl_mirekyou/kftl-mi-re-kyou-tag-statement-line'
import { KFTLMiReKyouNoneStatementLine } from '@/classes/kftl/kftl_mirekyou/kftl-mi-re-kyou-none-statement-line'
import { KFTLEndMiReKyouStatementLine } from '@/classes/kftl/kftl_mirekyou/kftl-end-mi-re-kyou-statement-line'
import { TextAreaInfo } from '@/classes/kftl/text-area-info'
import { i18n } from '@/i18n'

/**
 * 行のつながりは各クラスのコンストラクタに散っていて、どこにも一覧が無い。
 * KFTLStatement.generate_kftl_lines と同じ手順で1行ずつ組み立てて、
 * 次の行のコンストラクタを辿った結果が期待通りの並びかを見る。
 */
function build_statement_lines(
  line_texts: Array<string>,
  first_line_constructor: { (line_text: string, context: KFTLStatementLineContext): KFTLStatementLine } = (line_text, context) => new KFTLStartMiStatementLine(line_text, context),
): Array<KFTLStatementLine> {
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
    lines.push(next_constructor ? next_constructor(line_text, context) : first_line_constructor(line_text, context))
    prev_context = context
  }
  return lines
}

// リポストタスクのブロックは「先頭行が開始行」から組み立てる
function build_mi_re_kyou_lines(line_texts: Array<string>): Array<KFTLStatementLine> {
  return build_statement_lines(line_texts, (line_text, context) => new KFTLStartMiReKyouStatementLine(line_text, context, false))
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

/**
 * リポストタスク(「～～」)のブロック。
 * Mi と違ってタイトル行が無く、タグ行を板名の前にも後にも書ける。
 * Go 側(kftl_mirekyou.go)も同じ並びなので、崩したら両方直すこと
 */
describe('リポストタスクのブロック', () => {
  test('行順が 板名→見積開始→見積終了→期日→タグ→終了 になっている', () => {
    const lines = build_mi_re_kyou_lines(['～～', '仕事', '2025-03-20', '2025-03-21', '2025-03-22', '。今日中', '～～'])
    expect(lines[0]).toBeInstanceOf(KFTLStartMiReKyouStatementLine)
    expect(lines[1]).toBeInstanceOf(KFTLMiReKyouBoardNameStatementLine)
    expect(lines[2]).toBeInstanceOf(KFTLMiReKyouEstimateStartTimeStatementLine)
    expect(lines[3]).toBeInstanceOf(KFTLMiReKyouEstimateEndTimeStatementLine)
    expect(lines[4]).toBeInstanceOf(KFTLMiReKyouLimitTimeStatementLine)
    expect(lines[5]).toBeInstanceOf(KFTLMiReKyouTagStatementLine)
    expect(lines[6]).toBeInstanceOf(KFTLEndMiReKyouStatementLine)
  })

  // MiReKyou はタイトルを持たない(対象の記録をそのまま表示する)ので、
  // 板名の前にタイトル行を挟んではいけない
  test('タイトル行が無く、最初の非タグ行が板名になる', () => {
    const lines = build_mi_re_kyou_lines(['～～', '仕事', '～～'])
    expect(lines[1]).toBeInstanceOf(KFTLMiReKyouBoardNameStatementLine)
    expect(lines[1]).not.toBeInstanceOf(KFTLMiTitleStatementLine)
  })

  test('タグ行は項目の位置を消費しないので、板名の前に書いても次の非タグ行が板名になる', () => {
    const lines = build_mi_re_kyou_lines(['～～', '。今日中', '。重要', '仕事', '2025-03-20', '～～'])
    expect(lines[1]).toBeInstanceOf(KFTLMiReKyouTagStatementLine)
    expect(lines[2]).toBeInstanceOf(KFTLMiReKyouTagStatementLine)
    expect(lines[3]).toBeInstanceOf(KFTLMiReKyouBoardNameStatementLine)
    expect(lines[4]).toBeInstanceOf(KFTLMiReKyouEstimateStartTimeStatementLine)
    expect(lines[5]).toBeInstanceOf(KFTLEndMiReKyouStatementLine)
  })

  test('項目行の合間にタグ行を挟んでも次の非タグ行が次の項目になる', () => {
    const lines = build_mi_re_kyou_lines(['～～', '仕事', '。今日中', '2025-03-20', '～～'])
    expect(lines[1]).toBeInstanceOf(KFTLMiReKyouBoardNameStatementLine)
    expect(lines[2]).toBeInstanceOf(KFTLMiReKyouTagStatementLine)
    expect(lines[3]).toBeInstanceOf(KFTLMiReKyouEstimateStartTimeStatementLine)
  })

  test('開始行の次が「～～」なら空のブロックとして閉じる', () => {
    const lines = build_mi_re_kyou_lines(['～～', '～～'])
    expect(lines[1]).toBeInstanceOf(KFTLEndMiReKyouStatementLine)
  })

  test('板名だけ書いて途中で閉じられる', () => {
    const lines = build_mi_re_kyou_lines(['～～', '仕事', '～～'])
    expect(lines[2]).toBeInstanceOf(KFTLEndMiReKyouStatementLine)
  })

  // 項目行のあとの受け皿をタグ行にすると、その行が「次もタグ行」を指し続けるので、
  // 行ラベルの先読み(generate_line_label_data)が「タグ」を上限ぶん並べてしまう。
  // 受け皿はMiの期日行のあとと同じく「**********」の行にする
  test('項目を書き終えたあとの空行はタグ行ではなく「**********」の行になる', () => {
    const lines = build_mi_re_kyou_lines(['～～', '仕事', '2025-03-20', '2025-03-21', '2025-03-22', '', ''])
    expect(lines[5]).toBeInstanceOf(KFTLMiReKyouNoneStatementLine)
    expect(lines[6]).toBeInstanceOf(KFTLMiReKyouNoneStatementLine)
  })

  // 素の KFTLNoneStatementLine を受け皿にすると「～～」が閉じる行ではなく
  // 新しいブロックの開始行として解釈され、ブロックを閉じられなくなる
  test('空行を挟んでも「～～」で閉じられる', () => {
    const lines = build_mi_re_kyou_lines(['～～', '仕事', '2025-03-20', '2025-03-21', '2025-03-22', '', '～～'])
    expect(lines[6]).toBeInstanceOf(KFTLEndMiReKyouStatementLine)
  })

  test('空行を挟んだあとに書いた後置タグもリポストタスクのタグ行になる', () => {
    const lines = build_mi_re_kyou_lines(['～～', '仕事', '2025-03-20', '2025-03-21', '2025-03-22', '', '。今日中', '～～'])
    expect(lines[6]).toBeInstanceOf(KFTLMiReKyouTagStatementLine)
    expect(lines[7]).toBeInstanceOf(KFTLEndMiReKyouStatementLine)
  })

  test('閉じたあとはリポストタスクの入力は終わる', () => {
    const lines = build_mi_re_kyou_lines(['～～', '仕事', '～～', 'ただのメモ'])
    expect(lines[3]).not.toBeInstanceOf(KFTLMiReKyouBoardNameStatementLine)
    expect(lines[3]).not.toBeInstanceOf(KFTLMiReKyouEstimateStartTimeStatementLine)
    expect(lines[3]).not.toBeInstanceOf(KFTLMiReKyouTagStatementLine)
    expect(lines[3]).not.toBeInstanceOf(KFTLEndMiReKyouStatementLine)
  })

  // 「～」はWindowsのIMEがU+FF5E、macOS/iOSのIMEがU+301Cを出す。
  // 正規化を落とすと iOS からだけ記法が効かなくなる
  test('波ダッシュ(U+301C)で書いても開始行・終了行として扱う', () => {
    const lines = build_mi_re_kyou_lines(['〜〜', '仕事', '〜〜'])
    expect(lines[0]).toBeInstanceOf(KFTLStartMiReKyouStatementLine)
    expect(lines[2]).toBeInstanceOf(KFTLEndMiReKyouStatementLine)
  })

  test('ASCII の ~~ でも同じ並びになる', () => {
    const lines = build_mi_re_kyou_lines(['~~', '仕事', '。今日中', '~~'])
    expect(lines[1]).toBeInstanceOf(KFTLMiReKyouBoardNameStatementLine)
    expect(lines[2]).toBeInstanceOf(KFTLMiReKyouTagStatementLine)
    expect(lines[3]).toBeInstanceOf(KFTLEndMiReKyouStatementLine)
  })
})

/**
 * 行ラベルの先読み。
 * generate_line_label_data は「次の行のコンストラクタ」がある限り空行を組み立てて
 * ラベルを並べるので、自分自身を次に指す行があると先読み上限(50行)ぶん並ぶ。
 */
describe('リポストタスクの行ラベルの先読み', () => {
  function label_names(text: string): Array<string> {
    return new KFTLStatement(text).generate_line_label_data(new TextAreaInfo()).map(label_data => label_data.label)
  }

  const tag_label = i18n.global.t('KFTL_TAG_LABEL_TITLE')
  const none_label = i18n.global.t('KFTL_NONE_LABEL_TITLE')

  test('項目行のあとに「タグ」が並ばない', () => {
    const labels = label_names('メモ\n～～\n')
    expect(labels).not.toContain(tag_label)
    expect(labels[labels.length - 1]).toBe(none_label)
  })

  // 先読みの見た目はMiの期日行のあとと揃える
  test('期日行より先の先読みはすべて「**********」になる', () => {
    const labels = label_names('メモ\n～～\n')
    const limit_index = labels.indexOf(i18n.global.t('KFTL_MI_NO_LIMIT_TIME_TITLE'))
    expect(limit_index).toBeGreaterThan(0)
    const lookahead_after_limit = labels.slice(limit_index + 1)
    expect(lookahead_after_limit.length).toBeGreaterThan(1)
    expect(lookahead_after_limit.every(label => label === none_label)).toBe(true)
  })

  test('実際に書いた後置タグのラベルは「タグ」のまま', () => {
    const labels = label_names('メモ\n～～\n仕事\n2025-03-20\n2025-03-21\n2025-03-22\n。今日中\n～～\n')
    expect(labels.filter(label => label === tag_label)).toHaveLength(1)
  })
})

/**
 * 支出の行ラベルの先読み。
 * 支払い(品名と金額のペア)が繰り返す記法なので、先読みは「タイトル」と「金額」の交互で
 * 埋まるのが正しい。ブロックに留まる行を足したせいで、書いてもいないタグが
 * 上限(50行)ぶん並ばないことを固定する。
 */
describe('支出の行ラベルの先読み', () => {
  function label_names(text: string): Array<string> {
    return new KFTLStatement(text).generate_line_label_data(new TextAreaInfo()).map(label_data => label_data.label)
  }

  const tag_label = i18n.global.t('KFTL_TAG_LABEL_TITLE')
  const title_label = i18n.global.t('KFTL_NLOG_TITLE_TITLE')
  const amount_label = i18n.global.t('KFTL_NLOG_AMOUNT_LABEL_TITLE')

  test('金額のあとの先読みに「タグ」が並ばない', () => {
    const labels = label_names('ーん\nコンビニ\nおにぎり\n150\n')
    expect(labels).not.toContain(tag_label)
  })

  test('先読みは「タイトル」と「金額」の交互のまま', () => {
    const labels = label_names('ーん\nコンビニ\nおにぎり\n150\n')
    const lookahead = labels.slice(4)
    expect(lookahead.length).toBeGreaterThan(1)
    expect(lookahead.every(label => label === title_label || label === amount_label)).toBe(true)
  })

  test('実際に書いたタグ行のラベルは「タグ」のまま', () => {
    const labels = label_names('ーん\nコンビニ\nおにぎり\n150\n。食費\nお茶\n120\n')
    expect(labels.filter(label => label === tag_label)).toHaveLength(1)
  })
})
