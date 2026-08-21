import { describe, test, expect, beforeEach, afterEach, vi } from 'vitest'
import { parse_kftl_date_time } from '@/classes/kftl/kftl-date-time'

// Go の src/server/gkill/api/kftl/kftl_date_time_test.go と対のテーブル。
// 「今日」の基準を固定するため fake timers で 2026-08-20 21:30 に固定する。
describe('parse_kftl_date_time', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 7, 20, 21, 30, 0)) // 月は0始まり → 8月
  })
  afterEach(() => {
    vi.useRealTimers()
  })

  test('時刻のみ HH:mm は今日の年月日に載る（1月1日にならない）', () => {
    const d = parse_kftl_date_time('15:04')
    expect(d).not.toBeNull()
    expect(d!.getFullYear()).toBe(2026)
    expect(d!.getMonth()).toBe(7) // 8月
    expect(d!.getDate()).toBe(20)
    expect(d!.getHours()).toBe(15)
    expect(d!.getMinutes()).toBe(4)
  })

  test('時刻のみ HH:mm:ss', () => {
    const d = parse_kftl_date_time('15:04:05')
    expect(d).not.toBeNull()
    expect(d!.getDate()).toBe(20)
    expect(d!.getSeconds()).toBe(5)
  })

  test('年省略 M/D HH:mm は月日を尊重し年だけ今年で補完（Chromiumの2001年問題を踏まない）', () => {
    const d = parse_kftl_date_time('1/2 15:04')
    expect(d).not.toBeNull()
    expect(d!.getFullYear()).toBe(2026)
    expect(d!.getMonth()).toBe(0) // 1月
    expect(d!.getDate()).toBe(2)
    expect(d!.getHours()).toBe(15)
  })

  test('完全日時はそのまま解釈される', () => {
    const d = parse_kftl_date_time('2026-03-15 10:20:30')
    expect(d).not.toBeNull()
    expect(d!.getFullYear()).toBe(2026)
    expect(d!.getMonth()).toBe(2) // 3月
    expect(d!.getDate()).toBe(15)
    expect(d!.getHours()).toBe(10)
  })

  test('パースできない入力は null', () => {
    expect(parse_kftl_date_time('not a time')).toBeNull()
  })

  test('年またぎ境界: 時刻のみは前日推測をせず当日に補完', () => {
    vi.setSystemTime(new Date(2026, 0, 1, 0, 10, 0)) // 2026-01-01 00:10
    const d = parse_kftl_date_time('23:50')
    expect(d).not.toBeNull()
    expect(d!.getFullYear()).toBe(2026)
    expect(d!.getMonth()).toBe(0)
    expect(d!.getDate()).toBe(1)
    expect(d!.getHours()).toBe(23)
  })
})
