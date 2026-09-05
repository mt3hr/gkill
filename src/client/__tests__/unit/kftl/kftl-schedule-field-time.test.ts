import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'

// Mock @/i18n so all KFTL modules use our test i18n
vi.mock('@/i18n', () => ({ i18n }))

import { parse_schedule_field_time } from '@/classes/kftl/kftl-schedule-field-time'
import { KFTLStatement } from '@/classes/kftl/kftl-statement'

// Go の src/server/gkill/api/kftl/kftl_schedule_field_time_test.go と対のテーブル。
// 「今日」の基準を固定するため fake timers で 2026-08-20 21:30 に固定する。
describe('parse_schedule_field_time', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 7, 20, 21, 30, 0)) // 月は0始まり → 8月
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  test('完全な日時はそのまま読む', () => {
    const d = parse_schedule_field_time('2026-03-15 10:20')
    expect(d).not.toBeNull()
    expect(d!.getFullYear()).toBe(2026)
    expect(d!.getMonth()).toBe(2) // 3月
    expect(d!.getDate()).toBe(15)
    expect(d!.getHours()).toBe(10)
  })

  test('時刻のみは今日の年月日に載る', () => {
    const d = parse_schedule_field_time('18:00')
    expect(d).not.toBeNull()
    expect(d!.getDate()).toBe(20)
    expect(d!.getHours()).toBe(18)
  })

  test('空行と空白のみは未設定', () => {
    expect(parse_schedule_field_time('')).toBeNull()
    expect(parse_schedule_field_time('   ')).toBeNull()
  })

  // 日時として読めない行を一律に行エラーへ倒すと既存の書き方が広範に壊れるので、
  // ここは従来どおり未設定のまま。変えるのは「？」だけ。
  test('読めない行は未設定のままで投げない', () => {
    expect(parse_schedule_field_time('not a time')).toBeNull()
  })

  // 以前は接頭辞として剥がしたうえ、残りのパース失敗を未設定として握り潰していた。
  // 「？？」(繰り返しブロック)の書き損じが無音で消えるのはこの経路。
  test('関連時刻の接頭辞は例外にする', () => {
    for (const input of ['？18:00', '?18:00', '？', '?', '？？']) {
      expect(() => parse_schedule_field_time(input)).toThrow()
    }
  })
})

// Mi / MiReKyou の予定日時3欄すべてで「？」が不正行になること。
// 欄ごとに実装が分かれているので、1欄だけ直した取りこぼしをここで捕まえる。
describe('予定日時欄の「？」は不正行になる', () => {
  const cases: Array<{ name: string; text: string; invalid_line: number }> = [
    { name: 'Mi 見積開始', text: 'ーみ\nタスク\n仕事\n？18:00', invalid_line: 3 },
    { name: 'Mi 見積終了', text: 'ーみ\nタスク\n仕事\n\n？18:00', invalid_line: 4 },
    { name: 'Mi 期限', text: 'ーみ\nタスク\n仕事\n\n\n？18:00', invalid_line: 5 },
    { name: 'Mi ASCII接頭辞', text: 'ーみ\nタスク\n仕事\n?18:00', invalid_line: 3 },
    { name: 'MiReKyou 見積開始', text: 'メモ\n～～\n仕事\n？18:00\n～～', invalid_line: 3 },
    { name: 'MiReKyou 見積終了', text: 'メモ\n～～\n仕事\n\n？18:00\n～～', invalid_line: 4 },
    { name: 'MiReKyou 期限', text: 'メモ\n～～\n仕事\n\n\n？18:00\n～～', invalid_line: 5 },
  ]
  for (const c of cases) {
    test(c.name, async () => {
      const invalids = await new KFTLStatement(c.text).get_invalid_line_indexs()
      expect(invalids).toContain(c.invalid_line)
    })
  }
})

// 接頭辞なしの書き方は今までどおり通ること(禁止のとばっちりで壊れていないこと)。
describe('接頭辞なしの予定日時は今までどおり通る', () => {
  test('Mi の3欄が全部読まれ、不正行にならない', async () => {
    const text = 'ーみ\nタスク\n仕事\n2026-03-20 09:00\n2026-03-20 10:00\n2026-03-21 18:00'
    expect(await new KFTLStatement(text).get_invalid_line_indexs()).toEqual([])
  })
})
