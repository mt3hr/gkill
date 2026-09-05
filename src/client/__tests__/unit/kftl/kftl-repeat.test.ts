import { describe, test, expect, vi } from 'vitest'
import { i18n } from '../../helpers/setup-i18n'

// Mock @/i18n so all KFTL modules use our test i18n
vi.mock('@/i18n', () => ({ i18n }))

import {
    parse_repeat_condition,
    parse_repeat_count_or_until,
    parse_repeat_add_if_exists,
    parse_repeat_origin,
    occurrences_of,
    days_between,
    shift_days,
    new_repeat_spec,
    type RepeatSpec,
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

// ─── 候補日時 ─────────────────────────────────────────────────────────────────

function spec_of(cond_text: string, count: number, origin: Date, until: Date | null = null): RepeatSpec {
    const spec = new_repeat_spec(0)
    spec.cond = parse_repeat_condition(cond_text)
    spec.count = count
    spec.until = until
    spec.origin = origin
    return spec
}

function expect_times(got: Array<Date>, want: Array<Date>): void {
    expect(got.map((d) => d.getTime())).toEqual(want.map((d) => d.getTime()))
}

// 2026-09-02 は水曜。時刻はアンカーから取り、起点ちょうどは含めない。
describe('occurrences_of', () => {
    const origin = ymdhm(2026, 9, 2, 10, 0)

    test('毎日。アンカーが起点より後なら当日も入る', () => {
        const got = occurrences_of(spec_of('毎日', 3, origin), ymdhm(2026, 9, 2, 18, 0), origin)
        expect_times(got, [ymdhm(2026, 9, 2, 18, 0), ymdhm(2026, 9, 3, 18, 0), ymdhm(2026, 9, 4, 18, 0)])
    })

    test('毎日。アンカーが起点より前なら当日は落ちる', () => {
        const got = occurrences_of(spec_of('毎日', 3, origin), ymdhm(2026, 9, 2, 8, 0), origin)
        expect_times(got, [ymdhm(2026, 9, 3, 8, 0), ymdhm(2026, 9, 4, 8, 0), ymdhm(2026, 9, 5, 8, 0)])
    })

    test('曜日', () => {
        const got = occurrences_of(spec_of('金', 3, origin), ymdhm(2026, 9, 2, 18, 0), origin)
        expect_times(got, [ymdhm(2026, 9, 4, 18, 0), ymdhm(2026, 9, 11, 18, 0), ymdhm(2026, 9, 18, 18, 0)])
    })

    test('複数曜日は書いた順ではなく日付順', () => {
        const got = occurrences_of(spec_of('月水金', 4, origin), ymdhm(2026, 9, 2, 18, 0), origin)
        expect_times(got, [
            ymdhm(2026, 9, 2, 18, 0), ymdhm(2026, 9, 4, 18, 0),
            ymdhm(2026, 9, 7, 18, 0), ymdhm(2026, 9, 9, 18, 0),
        ])
    })

    test('N週おきは最初の一致を基準にする', () => {
        const got = occurrences_of(spec_of('6週金', 3, origin), ymdhm(2026, 9, 2, 18, 0), origin)
        expect_times(got, [ymdhm(2026, 9, 4, 18, 0), ymdhm(2026, 10, 16, 18, 0), ymdhm(2026, 11, 27, 18, 0)])
    })

    test('毎月N日', () => {
        const got = occurrences_of(spec_of('毎月15', 3, origin), ymdhm(2026, 9, 2, 18, 0), origin)
        expect_times(got, [ymdhm(2026, 9, 15, 18, 0), ymdhm(2026, 10, 15, 18, 0), ymdhm(2026, 11, 15, 18, 0)])
    })

    test('第N曜日', () => {
        const got = occurrences_of(spec_of('第2金', 3, origin), ymdhm(2026, 9, 2, 18, 0), origin)
        expect_times(got, [ymdhm(2026, 9, 11, 18, 0), ymdhm(2026, 10, 9, 18, 0), ymdhm(2026, 11, 13, 18, 0)])
    })

    test('最終曜日', () => {
        const got = occurrences_of(spec_of('最終金', 3, origin), ymdhm(2026, 9, 2, 18, 0), origin)
        expect_times(got, [ymdhm(2026, 9, 25, 18, 0), ymdhm(2026, 10, 30, 18, 0), ymdhm(2026, 11, 27, 18, 0)])
    })

    // 存在しない日はその月を飛ばす。最も近い日へ丸めると第4金・毎月30と重複する
    test('毎月31は31日の無い月を飛ばす', () => {
        const jan = ymdhm(2026, 1, 1, 0, 0)
        const got = occurrences_of(spec_of('毎月31', 3, jan), ymdhm(2026, 1, 1, 9, 0), jan)
        expect_times(got, [ymdhm(2026, 1, 31, 9, 0), ymdhm(2026, 3, 31, 9, 0), ymdhm(2026, 5, 31, 9, 0)])
    })

    test('第5金は第5金の無い月を飛ばす', () => {
        const jan = ymdhm(2026, 1, 1, 0, 0)
        const got = occurrences_of(spec_of('第5金', 3, jan), ymdhm(2026, 1, 1, 9, 0), jan)
        expect_times(got, [ymdhm(2026, 1, 30, 9, 0), ymdhm(2026, 5, 29, 9, 0), ymdhm(2026, 7, 31, 9, 0)])
    })

    test('起点ちょうどは含めない', () => {
        const got = occurrences_of(spec_of('毎日', 2, origin), ymdhm(2026, 9, 2, 10, 0), origin)
        expect_times(got, [ymdhm(2026, 9, 3, 10, 0), ymdhm(2026, 9, 4, 10, 0)])
    })

    test('終了日で止まる', () => {
        const until = new Date(2026, 8, 15, 23, 59, 59)
        const got = occurrences_of(spec_of('金', 0, origin, until), ymdhm(2026, 9, 2, 18, 0), origin)
        expect_times(got, [ymdhm(2026, 9, 4, 18, 0), ymdhm(2026, 9, 11, 18, 0)])
    })

    // 上限は黙って切り詰めず例外にする（部分確定するので大量生成の途中失敗が痛い）
    test('上限超過は例外', () => {
        const until = new Date(2036, 8, 2, 23, 59, 59) // 約10年ぶん = 3600件超
        expect(() => occurrences_of(spec_of('毎日', 0, origin, until), ymdhm(2026, 9, 2, 18, 0), origin)).toThrow()
    })
})

describe('日数のずらし', () => {
    test('days_between は暦日の差', () => {
        expect(days_between(ymdhm(2026, 9, 2, 18, 0), ymdhm(2026, 9, 4, 9, 0))).toBe(2)
        expect(days_between(ymdhm(2026, 9, 4, 9, 0), ymdhm(2026, 9, 2, 18, 0))).toBe(-2)
        expect(days_between(ymdhm(2026, 9, 2, 0, 0), ymdhm(2026, 9, 2, 23, 0))).toBe(0)
    })

    test('shift_days は壁時計時刻を保つ', () => {
        const shifted = shift_days(ymdhm(2026, 9, 2, 18, 30), 9)
        expect(shifted.getTime()).toBe(ymdhm(2026, 9, 11, 18, 30).getTime())
    })
})
