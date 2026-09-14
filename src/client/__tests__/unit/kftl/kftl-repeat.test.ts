import { describe, test, expect, vi } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'

// Mock @/i18n so all KFTL modules use our test i18n
vi.mock('@/i18n', () => ({ i18n }))

import {
    parse_repeat_condition,
    parse_repeat_count_or_until,
    parse_repeat_add_if_exists,
    parse_repeat_origin,
} from '@/classes/kftl/kftl_repeat/kftl-repeat-spec'

// Go の src/server/gkill/api/kftl/kftl_repeat_test.go と対のテーブル。

function ymdhm(y: number, m: number, d: number, hh: number, mm: number): Date {
    return new Date(y, m - 1, d, hh, mm, 0, 0)
}

describe('parse_repeat_condition', () => {
    const cases: Array<{ input: string; kind: string; weekdays: Array<number>; interval: number; nth: number; month_day: number }> = [
        { input: '毎日', kind: 'daily', weekdays: [], interval: 0, nth: 0, month_day: 0 },
        { input: 'daily', kind: 'daily', weekdays: [], interval: 0, nth: 0, month_day: 0 },
        { input: '金', kind: 'weekday', weekdays: [5], interval: 1, nth: 0, month_day: 0 },
        { input: 'fri', kind: 'weekday', weekdays: [5], interval: 1, nth: 0, month_day: 0 },
        { input: '月水金', kind: 'weekday', weekdays: [1, 3, 5], interval: 1, nth: 0, month_day: 0 },
        { input: 'mon,wed,fri', kind: 'weekday', weekdays: [1, 3, 5], interval: 1, nth: 0, month_day: 0 },
        { input: '2週月', kind: 'weekday', weekdays: [1], interval: 2, nth: 0, month_day: 0 },
        { input: '6週金', kind: 'weekday', weekdays: [5], interval: 6, nth: 0, month_day: 0 },
        { input: '6w fri', kind: 'weekday', weekdays: [5], interval: 6, nth: 0, month_day: 0 },
        { input: '毎月15', kind: 'month_day', weekdays: [], interval: 0, nth: 0, month_day: 15 },
        { input: 'monthly 15', kind: 'month_day', weekdays: [], interval: 0, nth: 0, month_day: 15 },
        { input: '第2金', kind: 'nth_weekday', weekdays: [5], interval: 0, nth: 2, month_day: 0 },
        { input: '2nd fri', kind: 'nth_weekday', weekdays: [5], interval: 0, nth: 2, month_day: 0 },
        { input: '最終金', kind: 'nth_weekday', weekdays: [5], interval: 0, nth: -1, month_day: 0 },
        { input: 'last fri', kind: 'nth_weekday', weekdays: [5], interval: 0, nth: -1, month_day: 0 },
    ]
    for (const c of cases) {
        test(c.input, () => {
            const cond = parse_repeat_condition(c.input)
            expect(cond.kind).toBe(c.kind)
            expect(cond.weekdays).toEqual(c.weekdays)
            expect(cond.week_interval).toBe(c.interval)
            expect(cond.nth).toBe(c.nth)
            expect(cond.month_day).toBe(c.month_day)
        })
    }

    // 「N週」は基準の週を1つに決めないといけないので、曜日を複数書けない
    test('読めない条件は例外', () => {
        for (const input of ['', '   ', 'きのう', '2週月水', '毎月0', '毎月32', '第6金', '6th fri', 'monday', '毎月']) {
            expect(() => parse_repeat_condition(input), input).toThrow()
        }
    })
})

describe('parse_repeat_count_or_until', () => {
    test('整数は回数', () => {
        const got = parse_repeat_count_or_until('3')
        expect(got.count).toBe(3)
        expect(got.until).toBeNull()
    })

    // 日付のみの終了日はその日の終わりまで伸ばす。伸ばさないと当日18:00が範囲外になる
    test('日付のみの終了日は23:59:59まで伸びる', () => {
        const got = parse_repeat_count_or_until('2026-12-31')
        expect(got.count).toBe(0)
        expect(got.until).not.toBeNull()
        expect(got.until!.getTime()).toBe(new Date(2026, 11, 31, 23, 59, 59).getTime())
    })

    test('時刻つきの終了日はそのまま', () => {
        const got = parse_repeat_count_or_until('2026-12-31 12:00')
        expect(got.until!.getTime()).toBe(new Date(2026, 11, 31, 12, 0, 0).getTime())
    })

    test('範囲外の回数と読めない行は例外', () => {
        for (const input of ['', '0', '-1', '1001', 'みっつ']) {
            expect(() => parse_repeat_count_or_until(input), input).toThrow()
        }
    })
})

describe('parse_repeat_add_if_exists', () => {
    test('既定は追加しない', () => {
        for (const input of ['', '  ', 'no', 'NO', 'いいえ']) {
            expect(parse_repeat_add_if_exists(input), input).toBe(false)
        }
    })
    test('yes は追加する', () => {
        for (const input of ['yes', 'YES', 'はい']) {
            expect(parse_repeat_add_if_exists(input), input).toBe(true)
        }
    })
    test('yes/no 以外は例外', () => {
        expect(() => parse_repeat_add_if_exists('maybe')).toThrow()
    })
})

describe('parse_repeat_origin', () => {
    test('空は null', () => {
        expect(parse_repeat_origin('')).toBeNull()
    })
    test('日時として読む', () => {
        const got = parse_repeat_origin('2026-09-12 18:00')
        expect(got!.getTime()).toBe(ymdhm(2026, 9, 12, 18, 0).getTime())
    })
    test('読めない起点は例外', () => {
        expect(() => parse_repeat_origin('きのう')).toThrow()
    })
})

// 候補日時の計算（occurrences_of）・日数のずらし・上限は Go 側だけが持つ（ADR-0507）。
// 対のテストは src/server/gkill/api/kftl/kftl_repeat_test.go。
