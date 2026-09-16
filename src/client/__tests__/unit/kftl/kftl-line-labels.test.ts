import { describe, test, expect } from 'vitest'
import '../../helpers/setup-i18n'
import { KFTLStatement } from '@/classes/kftl/kftl-statement'
import { TextAreaInfo } from '@/classes/kftl/text-area-info'
import { i18n } from '@/i18n'

/**
 * 直線的な型（数値記録・気分・打刻6種・ブックマーク・テキスト・区切り・関連時刻）の行ラベル。
 *
 * TS 側の分類器は行ラベルを出すためだけにある（ADR-0507）。Mi / リポストタスク / 支出 / 繰り返しは
 * kftl-statement.test.ts が並びと先読みを固定しているが、残りの型は「接頭辞の判定」
 * （kftl-type-detection.test.ts）しか見ておらず、開始行から次の行へ辿るコンストラクタの連鎖と
 * 値の行の「変な○○」判定はどこにも無かった。接頭辞を Go 側と揃えて足したときに
 * ラベルの連鎖だけが古いまま残る（＝ラベルが嘘になる）のをここで止める。
 */
describe('直線的な型の行ラベル', () => {
  const t = (key: string): string => i18n.global.t(key)

  function written_labels(text: string): Array<string> {
    const line_count = text.split('\n').length
    return new KFTLStatement(text).generate_line_label_data(new TextAreaInfo()).slice(0, line_count).map(label_data => label_data.label)
  }

  test.each([
    {
      name: '数値記録: 開始→タイトル→数値',
      text: 'ーか\n体重\n65.5',
      labels: ['KFTL_KC_LABEL_TITLE', 'KFTL_KC_TITLE_TITLE', 'KFTL_KC_NUM_VALUE_TITLE'],
    },
    {
      name: '数値記録: 数値でない行は「変な数値」',
      text: 'ーか\n体重\nabc',
      labels: ['KFTL_KC_LABEL_TITLE', 'KFTL_KC_TITLE_TITLE', 'KFTL_KC_INVALID_NUM_VALUE_TITLE'],
    },
    {
      name: '気分: 開始→気分値',
      text: 'ーら\n7',
      labels: ['KFTL_LANTANA_LABEL_TITLE', 'KFTL_LANTANA_MOOD_VALUE_TITLE'],
    },
    {
      name: '気分: 0〜10 の外は「変な気分値」',
      text: 'ーら\n11',
      labels: ['KFTL_LANTANA_LABEL_TITLE', 'KFTL_LANTANA_INVALID_MOOD_VALUE_TITLE'],
    },
    {
      name: '打刻(完全): 開始→タイトル→開始日時→終了日時',
      text: 'ーち\n会議\n2025-03-15 10:00\n2025-03-15 11:30',
      labels: ['KFTL_TIMEIS_TIMEIS_LABEL_TITLE', 'KFTL_TIMEIS_TITLE_LABEL_TITLE', 'KFTL_TIMEIS_START_TIME_LABEL_TITLE', 'KFTL_TIMEIS_END_TIME_LABEL_TITLE'],
    },
    {
      name: '打刻開始: 開始→「開始」',
      text: 'ーた\n作業',
      labels: ['KFTL_TIMEIS_TIMEIS_LABEL_TITLE', 'KFTL_TIMEIS_TIMEIS_START_LABEL_TITLE'],
    },
    {
      name: '打刻終了: 開始→「終了」',
      text: 'ーえ\n作業',
      labels: ['KFTL_TIMEIS_TIMEIS_LABEL_TITLE', 'KFTL_TIMEIS_TIMEIS_END_LABEL_TITLE'],
    },
    {
      name: '打刻終了(存在時): 開始→「終了」',
      text: 'ーいえ\n作業',
      labels: ['KFTL_TIMEIS_TIMEIS_LABEL_TITLE', 'KFTL_TIMEIS_TIMEIS_END_LABEL_TITLE'],
    },
    {
      name: 'タグで打刻終了: 開始→「終了」',
      text: 'ーたえ\n仕事',
      labels: ['KFTL_TIMEIS_TIMEIS_LABEL_TITLE', 'KFTL_TIMEIS_TIMEIS_END_LABEL_TITLE'],
    },
    {
      name: 'タグで打刻終了(存在時): 開始→「終了」',
      text: 'ーいたえ\n仕事',
      labels: ['KFTL_TIMEIS_TIMEIS_LABEL_TITLE', 'KFTL_TIMEIS_TIMEIS_END_LABEL_TITLE'],
    },
    {
      name: 'ブックマーク: 開始→URL→タイトル（URL が先）',
      text: 'ーう\nhttps://example.com\nお気に入り',
      labels: ['KFTL_URLOG_URLOG_LABEL_TITLE', 'KFTL_URLOG_URL_LABEL_TITLE', 'KFTL_URLOG_TITLE_LABEL_TITLE'],
    },
    {
      name: 'テキスト: 開始行と終了行だけラベルが付き、本文行は空',
      text: 'ーー\n1行目\n2行目\nーー',
      labels: ['KFTL_TEXT_START_LABEL_TITLE', '', '', 'KFTL_TEXT_END_LABEL_TITLE'],
    },
    {
      name: '区切り: 「、」と「、、」で別のラベル',
      text: 'メモ\n、\nメモ\n、、\nメモ',
      labels: ['KFTL_KMEMO_LABEL_TITLE', 'KFTL_SPLIT_LABEL_TITLE', 'KFTL_KMEMO_LABEL_TITLE', 'KFTL_SPLIT_APPEND_TIME_LABEL_TITLE', 'KFTL_KMEMO_LABEL_TITLE'],
    },
    {
      name: '関連時刻: 読める日時は「日時」、読めなければ「変な日時」',
      text: '？2025-03-15 14:30\nメモ\n？あした',
      labels: ['KFTL_RELATED_TIME_TITLE', 'KFTL_KMEMO_LABEL_TITLE', 'KFTL_INVALID_RELATED_TIME_TITLE'],
    },
    {
      // Go 側も同じ行を「none-state の非空行」として入力エラーにする（ピンクになる）。
      // 続きを書くには「、」で区切る
      name: 'タグ: メモに付けたタグ行のあとに続けて書いた行は受け皿（「**********」）',
      text: 'メモ\n。日記\nつづき',
      labels: ['KFTL_KMEMO_LABEL_TITLE', 'KFTL_TAG_LABEL_TITLE', 'KFTL_NONE_LABEL_TITLE'],
    },
    {
      name: 'タグ: 「、」で区切れば次のメモになる',
      text: 'メモ\n。日記\n、\nつづき',
      labels: ['KFTL_KMEMO_LABEL_TITLE', 'KFTL_TAG_LABEL_TITLE', 'KFTL_SPLIT_LABEL_TITLE', 'KFTL_KMEMO_LABEL_TITLE'],
    },
    {
      name: 'タグ: 先頭に書いたタグ行のあとはメモ（あとに書いた記録に付く）',
      text: '。日記\nメモ',
      labels: ['KFTL_TAG_LABEL_TITLE', 'KFTL_KMEMO_LABEL_TITLE'],
    },
  ])('$name', ({ text, labels }) => {
    expect(written_labels(text)).toStrictEqual(labels.map(key => (key === '' ? '' : t(key))))
  })

  // ASCII 接頭辞は日本語と同じ連鎖になる（非日本語ロケール向け。対応表は kftl-prefixes.ts）
  test.each([
    { ja: 'ーか\n体重\n65.5', ascii: '/num\n体重\n65.5' },
    { ja: 'ーら\n7', ascii: '/mood\n7' },
    { ja: 'ーち\n会議\n2025-03-15 10:00\n2025-03-15 11:30', ascii: '/timeis\n会議\n2025-03-15 10:00\n2025-03-15 11:30' },
    { ja: 'ーた\n作業', ascii: '/start\n作業' },
    { ja: 'ーえ\n作業', ascii: '/end\n作業' },
    { ja: 'ーいえ\n作業', ascii: '/end?\n作業' },
    { ja: 'ーたえ\n仕事', ascii: '/endt\n仕事' },
    { ja: 'ーいたえ\n仕事', ascii: '/endt?\n仕事' },
    { ja: 'ーう\nhttps://example.com\nお気に入り', ascii: '/url\nhttps://example.com\nお気に入り' },
    { ja: 'ーー\n本文\nーー', ascii: '--\n本文\n--' },
    { ja: 'メモ\n、\nメモ\n、、\nメモ', ascii: 'メモ\n,\nメモ\n,,\nメモ' },
    { ja: '？2025-03-15\nメモ\n。日記', ascii: '?2025-03-15\nメモ\n#日記' },
  ])('ASCII 接頭辞 $ascii は日本語と同じラベル列', ({ ja, ascii }) => {
    expect(written_labels(ascii)).toStrictEqual(written_labels(ja))
  })
})
