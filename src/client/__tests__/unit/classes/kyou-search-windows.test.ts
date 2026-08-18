/**
 * 検索の期間分割の境界計算。
 *
 * ここを間違えると **エラーも出ないまま検索結果が欠ける / 重複する**。
 * サーバは期間を秒精度(time.Unix())で比べるので、境界のずらし幅も秒でないといけない。
 */
import { describe, test, expect } from 'vitest'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import {
    apply_search_window,
    build_search_windows,
    can_split_search_into_windows,
} from '@/classes/kyou-search-windows'

function make_query(overrides: Partial<FindKyouQuery> = {}): FindKyouQuery {
    const query = new FindKyouQuery()
    query.calendar_start_date = new Date('2020-01-01T00:00:00+09:00')
    query.calendar_end_date = new Date('2026-08-18T00:00:00+09:00')
    Object.assign(query, overrides)
    return query
}

describe('検索の期間分割', () => {
    test('窓は新しい順に並ぶ', () => {
        const windows = build_search_windows(make_query())
        expect(windows).not.toBeNull()
        const list = windows as NonNullable<typeof windows>
        expect(list.length).toBeGreaterThan(1)
        for (let i = 1; i < list.length; i++) {
            expect(list[i].end.getTime()).toBeLessThan(list[i - 1].end.getTime())
        }
    })

    // 隙間があると記録が消え、重なると同じ記録が2回出る。
    // サーバは秒へ切り捨てて比較するので、境界は「秒」で見て隣接していること
    test('隣り合う窓は秒精度で重ならず、隙間も空かない', () => {
        const list = build_search_windows(make_query()) as NonNullable<ReturnType<typeof build_search_windows>>
        for (let i = 1; i < list.length; i++) {
            const previous_start_second = Math.floor((list[i - 1].start as Date).getTime() / 1000)
            const current_end_second = Math.floor(list[i].end.getTime() / 1000)
            expect(current_end_second, '窓' + i + 'の上限が前の窓の下限と重なっている').toBe(previous_start_second - 1)
        }
    })

    test('最初の窓の上限と最後の窓の下限は、元の期間と一致する', () => {
        const query = make_query()
        const list = build_search_windows(query) as NonNullable<ReturnType<typeof build_search_windows>>
        expect(list[0].end.getTime()).toBe((query.calendar_end_date as Date).getTime())
        expect((list[list.length - 1].start as Date).getTime()).toBe((query.calendar_start_date as Date).getTime())
    })

    test('下限が無ければ最後の窓も下限なしで閉じる', () => {
        const query = make_query({ calendar_start_date: null })
        const list = build_search_windows(query) as NonNullable<ReturnType<typeof build_search_windows>>
        expect(list[list.length - 1].start).toBeNull()
    })

    test('1窓に収まる短い期間は分割しない', () => {
        const query = make_query({ calendar_start_date: new Date('2026-08-17T00:00:00+09:00') })
        expect(build_search_windows(query)).toBeNull()
    })

    test('下限が上限より後なら分割しない', () => {
        const query = make_query({ calendar_start_date: new Date('2027-01-01T00:00:00+09:00') })
        expect(build_search_windows(query)).toBeNull()
    })

    // 分割すると結果が変わる条件。ここを緩めると静かに壊れる
    describe('分割してはいけない条件', () => {
        const cases: Array<[string, Partial<FindKyouQuery>]> = [
            ['mi板', { for_mi: true }],
            ['画像のみ', { is_image_only: true }],
            ['実行中(plaing)', { plaing_time: new Date('2026-08-18T00:00:00+09:00') }],
            ['TimeIs絞り込み(words)', { timeis_words: ['作業'] }],
            ['TimeIs絞り込み(not_words)', { timeis_not_words: ['作業'] }],
            ['地図(緯度)', { map_latitude: 35 }],
            ['地図(経度)', { map_longitude: 139 }],
            ['地図(半径)', { map_radius: 1 }],
            ['上限なし', { calendar_end_date: null }],
        ]
        for (const [name, overrides] of cases) {
            test(name, () => {
                const query = make_query(overrides)
                expect(can_split_search_into_windows(query), name + ' で分割してしまっている').toBe(false)
                expect(build_search_windows(query)).toBeNull()
            })
        }
    })

    test('ワード検索は分割してよい(レコード単体で合否が決まる)', () => {
        const query = make_query({ words: ['メモ'] })
        expect(can_split_search_into_windows(query)).toBe(true)
    })

    test('apply_search_windowは元のクエリを変更しない', () => {
        const query = make_query()
        const original_start = query.calendar_start_date
        const original_end = query.calendar_end_date
        const windowed = apply_search_window(query, {
            start: new Date('2026-08-01T00:00:00+09:00'),
            end: new Date('2026-08-18T00:00:00+09:00'),
        })
        expect(query.calendar_start_date).toBe(original_start)
        expect(query.calendar_end_date).toBe(original_end)
        expect((windowed.calendar_start_date as Date).getTime()).toBe(new Date('2026-08-01T00:00:00+09:00').getTime())
        // query_id は列の同一性なので、窓を当てても変わってはいけない
        expect(windowed.query_id).toBe(query.query_id)
    })
})
