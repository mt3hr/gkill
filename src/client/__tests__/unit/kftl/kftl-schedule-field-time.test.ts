import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'

// Mock @/i18n so all KFTL modules use our test i18n
vi.mock('@/i18n', () => ({ i18n }))

import { parse_schedule_field_time } from '@/classes/kftl/kftl-schedule-field-time'

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

// 予定日時3欄で「？」が不正行になること・接頭辞なしが通ることは、判定を持つ Go 側の
// kftl_schedule_field_time_test.go（TestStatement_MiScheduleFieldsRejectRelatedTimePrefix /
// TestStatement_MiReKyouScheduleFieldsRejectRelatedTimePrefix / TestStatement_MiScheduleFieldsAcceptPlainDateTime）
// が固定する。TS 側は行ラベルの「不正な期限」表示に使う parse_schedule_field_time だけを持つ（ADR-0507）。
