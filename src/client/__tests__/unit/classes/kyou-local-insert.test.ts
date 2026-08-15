/**
 * 追加されたKyouを再検索せずに列へ差し込むための、判定と整列のテスト。
 *
 * ここが崩れると例外もエラーも出ず、絞り込み中の列に一致しない行が黙って出る
 * (あるいは一致する行が出ない)。意味論の出どころは
 * `src/server/gkill/api/find_filter.go` で、対になるテストは
 * `kyou-local-insert-mi-parity.test.ts`。
 */
import { describe, expect, it } from 'vitest'

import {
    apply_mi_projection,
    can_decide_kyou_locally,
    can_decide_query_locally,
    compare_kyou_for_query,
    decide_local_insert,
    does_kyou_match_query,
    find_insert_index,
    insert_kyou_sorted,
} from '@/classes/kyou-local-insert'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { MiCheckState } from '@/classes/api/find_query/mi-check-state'
import { MiSortType } from '@/classes/api/find_query/mi-sort-type'
import type { Kyou } from '@/classes/datas/kyou'
import type { Mi } from '@/classes/datas/mi'

interface FakeKyouOptions {
    id?: string
    data_type?: string
    rep_name?: string
    related_time?: Date
    create_time?: Date
    update_time?: Date
    is_deleted?: boolean
    tags?: Array<string>
    typed_mi?: Partial<Mi> | null
    typed_mirekyou?: Partial<Mi> | null
}

function make_kyou(options?: FakeKyouOptions): Kyou {
    const create_time = options?.create_time ?? new Date('2026-08-01T00:00:00.000Z')
    return {
        id: options?.id ?? 'kyou-1',
        data_type: options?.data_type ?? 'kmemo',
        rep_name: options?.rep_name ?? 'rep_a',
        related_time: options?.related_time ?? create_time,
        create_time: create_time,
        update_time: options?.update_time ?? create_time,
        is_deleted: options?.is_deleted ?? false,
        attached_tags: (options?.tags ?? []).map(tag => ({ tag: tag })),
        typed_mi: options?.typed_mi ?? null,
        typed_mirekyou: options?.typed_mirekyou ?? null,
    } as unknown as Kyou
}

function make_mi(options?: Partial<Mi>): Partial<Mi> {
    return {
        board_name: 'board_a',
        is_checked: false,
        create_time: new Date('2026-08-01T00:00:00.000Z'),
        update_time: new Date('2026-08-01T00:00:00.000Z'),
        limit_time: null,
        estimate_start_time: null,
        estimate_end_time: null,
        ...options,
    }
}

/** タグ絞り込みを使わない素のクエリ。既定は tags/reps が [] = 0件なので明示的に外す */
function make_query(overrides?: Partial<FindKyouQuery>): FindKyouQuery {
    const query = new FindKyouQuery()
    query.tags = null
    query.reps = null
    Object.assign(query, overrides)
    return query
}

/** ローカル時刻の時分秒からUnix秒を作る(時間帯フィルタの指定はローカル時刻で解釈される) */
function local_second_of_day(hours: number, minutes: number): number {
    return new Date(2026, 0, 1, hours, minutes, 0).getTime() / 1000
}

describe('can_decide_query_locally', () => {
    it('素の条件は判定できる', () => {
        expect(can_decide_query_locally(make_query()).ok).toBe(true)
    })

    it('本文検索が指定されていたら判定できない', () => {
        expect(can_decide_query_locally(make_query({ words: [] })).ok).toBe(false)
        expect(can_decide_query_locally(make_query({ not_words: ['x'] })).ok).toBe(false)
    })

    it('timeis_words は空配列でも判定できない(「任意のTimeIsに覆われたKyou」という有効な指定)', () => {
        expect(can_decide_query_locally(make_query({ timeis_words: [] })).ok).toBe(false)
        expect(can_decide_query_locally(make_query({ timeis_not_words: [] })).ok).toBe(false)
    })

    it('地図は3値そろったときだけ判定できない', () => {
        expect(can_decide_query_locally(make_query({ map_latitude: 35 })).ok).toBe(true)
        expect(can_decide_query_locally(make_query({ map_latitude: 35, map_longitude: 139 })).ok).toBe(true)
        expect(can_decide_query_locally(make_query({ map_latitude: 35, map_longitude: 139, map_radius: 1 })).ok).toBe(false)
    })

    it('plaing_time / is_image_only は判定できない', () => {
        expect(can_decide_query_locally(make_query({ plaing_time: new Date() })).ok).toBe(false)
        expect(can_decide_query_locally(make_query({ is_image_only: true })).ok).toBe(false)
    })

    it('rep_types は null か for_mi の [mi] だけ判定できる', () => {
        expect(can_decide_query_locally(make_query({ rep_types: null })).ok).toBe(true)
        expect(can_decide_query_locally(make_query({ for_mi: true, rep_types: ['mi'] })).ok).toBe(true)
        expect(can_decide_query_locally(make_query({ for_mi: false, rep_types: ['mi'] })).ok).toBe(false)
        expect(can_decide_query_locally(make_query({ rep_types: ['kmemo'] })).ok).toBe(false)
        expect(can_decide_query_locally(make_query({ rep_types: [] })).ok).toBe(false)
    })
})

describe('can_decide_kyou_locally', () => {
    it('非mi列では複数行になる型を判定できない', () => {
        const query = make_query({ for_mi: false })
        expect(can_decide_kyou_locally(make_kyou({ data_type: 'mi_create' }), query).ok).toBe(false)
        expect(can_decide_kyou_locally(make_kyou({ data_type: 'mirekyou_create' }), query).ok).toBe(false)
        expect(can_decide_kyou_locally(make_kyou({ data_type: 'timeis_start' }), query).ok).toBe(false)
    })

    it('非mi列でも単一行の型は判定できる', () => {
        const query = make_query({ for_mi: false })
        expect(can_decide_kyou_locally(make_kyou({ data_type: 'kmemo' }), query).ok).toBe(true)
        expect(can_decide_kyou_locally(make_kyou({ data_type: 'urlog' }), query).ok).toBe(true)
    })

    it('for_mi列は常に1行なので判定できる', () => {
        const query = make_query({ for_mi: true })
        expect(can_decide_kyou_locally(make_kyou({ data_type: 'mi_create' }), query).ok).toBe(true)
        expect(can_decide_kyou_locally(make_kyou({ data_type: 'kmemo' }), query).ok).toBe(true)
    })
})

describe('does_kyou_match_query - タグ', () => {
    it('tags が null ならタグを見ない', () => {
        expect(does_kyou_match_query(make_kyou({ tags: [] }), make_query({ tags: null }))).toBe(true)
        expect(does_kyou_match_query(make_kyou({ tags: ['a'] }), make_query({ tags: null }))).toBe(true)
    })

    it('tags が空配列なら0件', () => {
        expect(does_kyou_match_query(make_kyou({ tags: ['a'] }), make_query({ tags: [], tags_and: false }))).toBe(false)
        expect(does_kyou_match_query(make_kyou({ tags: [] }), make_query({ tags: [], tags_and: true }))).toBe(false)
    })

    it('OR は完全一致・大小無視。部分一致はしない', () => {
        expect(does_kyou_match_query(make_kyou({ tags: ['foo'] }), make_query({ tags: ['Foo'] }))).toBe(true)
        expect(does_kyou_match_query(make_kyou({ tags: ['foobar'] }), make_query({ tags: ['foo'] }))).toBe(false)
    })

    it('「no tags」はタグ0個のKyouに一致する', () => {
        expect(does_kyou_match_query(make_kyou({ tags: [] }), make_query({ tags: ['no tags'] }))).toBe(true)
        expect(does_kyou_match_query(make_kyou({ tags: ['a'] }), make_query({ tags: ['no tags'] }))).toBe(false)
    })

    it('AND は全部のタグ名を持っているときだけ一致する', () => {
        const query = make_query({ tags: ['a', 'b'], tags_and: true })
        expect(does_kyou_match_query(make_kyou({ tags: ['A'] }), query)).toBe(false)
        expect(does_kyou_match_query(make_kyou({ tags: ['A', 'B'] }), query)).toBe(true)
        expect(does_kyou_match_query(make_kyou({ tags: ['A', 'B', 'C'] }), query)).toBe(true)
    })

    it('AND に「no tags」と他のタグを混ぜると成立しない', () => {
        const query = make_query({ tags: ['no tags', 'a'], tags_and: true })
        expect(does_kyou_match_query(make_kyou({ tags: [] }), query)).toBe(false)
        expect(does_kyou_match_query(make_kyou({ tags: ['a'] }), query)).toBe(false)
    })

    it('hide_tags はチェックされていないときだけ隠す', () => {
        const kyou = make_kyou({ tags: ['x', 'secret'] })
        expect(does_kyou_match_query(kyou, make_query({ tags: ['x'], hide_tags: ['secret'] }))).toBe(false)
        expect(does_kyou_match_query(kyou, make_query({ tags: ['x', 'secret'], hide_tags: ['secret'] }))).toBe(true)
    })

    it('hide_tags の照合は大小無視、チェック済み判定は大小区別', () => {
        const kyou = make_kyou({ tags: ['x', 'secret'] })
        // hide_tags 側は大小無視で当たる
        expect(does_kyou_match_query(kyou, make_query({ tags: ['x'], hide_tags: ['Secret'] }))).toBe(false)
        // チェック済み判定は大小区別なので Secret では外れず、隠されたまま
        expect(does_kyou_match_query(kyou, make_query({ tags: ['x', 'Secret'], hide_tags: ['Secret'] }))).toBe(false)
    })

    it('tags が null なら hide_tags は適用しない', () => {
        const kyou = make_kyou({ tags: ['secret'] })
        expect(does_kyou_match_query(kyou, make_query({ tags: null, hide_tags: ['secret'] }))).toBe(true)
    })
})

describe('does_kyou_match_query - rep / カレンダー / 時間帯', () => {
    it('reps は null=不使用 / []=0件 / それ以外は完全一致', () => {
        const kyou = make_kyou({ rep_name: 'rep_a' })
        expect(does_kyou_match_query(kyou, make_query({ reps: null }))).toBe(true)
        expect(does_kyou_match_query(kyou, make_query({ reps: [] }))).toBe(false)
        expect(does_kyou_match_query(kyou, make_query({ reps: ['rep_a'] }))).toBe(true)
        expect(does_kyou_match_query(kyou, make_query({ reps: ['rep_b'] }))).toBe(false)
    })

    it('カレンダーは両端を含む', () => {
        const start = new Date('2026-08-01T00:00:00.000Z')
        const end = new Date('2026-08-31T23:59:59.000Z')
        const query = make_query({ calendar_start_date: start, calendar_end_date: end })
        expect(does_kyou_match_query(make_kyou({ related_time: start }), query)).toBe(true)
        expect(does_kyou_match_query(make_kyou({ related_time: end }), query)).toBe(true)
        expect(does_kyou_match_query(make_kyou({ related_time: new Date(start.getTime() - 1) }), query)).toBe(false)
        expect(does_kyou_match_query(make_kyou({ related_time: new Date(end.getTime() + 1) }), query)).toBe(false)
    })

    it('week_of_days が null なら曜日を制限しない', () => {
        // 2026-08-03 は月曜
        const monday = new Date(2026, 7, 3, 12, 0, 0)
        const query = make_query({
            period_of_time_start_time_second: local_second_of_day(0, 0),
            period_of_time_end_time_second: local_second_of_day(23, 59),
            period_of_time_week_of_days: null,
        })
        expect(does_kyou_match_query(make_kyou({ related_time: monday }), query)).toBe(true)
    })

    it('week_of_days が空配列なら0件、全7曜日なら制限なし', () => {
        const monday = new Date(2026, 7, 3, 12, 0, 0)
        expect(does_kyou_match_query(make_kyou({ related_time: monday }), make_query({ period_of_time_week_of_days: [] }))).toBe(false)
        expect(does_kyou_match_query(make_kyou({ related_time: monday }), make_query({ period_of_time_week_of_days: [0, 1, 2, 3, 4, 5, 6] }))).toBe(true)
    })

    it('week_of_days は指定した曜日だけ通す', () => {
        const monday = new Date(2026, 7, 3, 12, 0, 0)
        expect(does_kyou_match_query(make_kyou({ related_time: monday }), make_query({ period_of_time_week_of_days: [1] }))).toBe(true)
        expect(does_kyou_match_query(make_kyou({ related_time: monday }), make_query({ period_of_time_week_of_days: [2] }))).toBe(false)
    })

    it('時間帯は両端を含み、start > end は夜跨ぎのORになる', () => {
        const query = make_query({
            period_of_time_start_time_second: local_second_of_day(9, 0),
            period_of_time_end_time_second: local_second_of_day(17, 0),
        })
        expect(does_kyou_match_query(make_kyou({ related_time: new Date(2026, 7, 3, 9, 0, 0) }), query)).toBe(true)
        expect(does_kyou_match_query(make_kyou({ related_time: new Date(2026, 7, 3, 17, 0, 0) }), query)).toBe(true)
        expect(does_kyou_match_query(make_kyou({ related_time: new Date(2026, 7, 3, 8, 59, 59) }), query)).toBe(false)

        const night_query = make_query({
            period_of_time_start_time_second: local_second_of_day(22, 0),
            period_of_time_end_time_second: local_second_of_day(5, 0),
        })
        expect(does_kyou_match_query(make_kyou({ related_time: new Date(2026, 7, 3, 23, 0, 0) }), night_query)).toBe(true)
        expect(does_kyou_match_query(make_kyou({ related_time: new Date(2026, 7, 3, 3, 0, 0) }), night_query)).toBe(true)
        expect(does_kyou_match_query(make_kyou({ related_time: new Date(2026, 7, 3, 12, 0, 0) }), night_query)).toBe(false)
    })

    it('時間帯は片側だけの指定でも効く', () => {
        const start_only = make_query({ period_of_time_start_time_second: local_second_of_day(9, 0) })
        expect(does_kyou_match_query(make_kyou({ related_time: new Date(2026, 7, 3, 10, 0, 0) }), start_only)).toBe(true)
        expect(does_kyou_match_query(make_kyou({ related_time: new Date(2026, 7, 3, 8, 0, 0) }), start_only)).toBe(false)

        const end_only = make_query({ period_of_time_end_time_second: local_second_of_day(17, 0) })
        expect(does_kyou_match_query(make_kyou({ related_time: new Date(2026, 7, 3, 16, 0, 0) }), end_only)).toBe(true)
        expect(does_kyou_match_query(make_kyou({ related_time: new Date(2026, 7, 3, 18, 0, 0) }), end_only)).toBe(false)
    })

    it('削除済みは一致しない', () => {
        expect(does_kyou_match_query(make_kyou({ is_deleted: true }), make_query())).toBe(false)
    })
})

describe('does_kyou_match_query - mi', () => {
    const base_query = () => make_query({
        for_mi: true,
        mi_check_state: MiCheckState.all,
        mi_sort_type: MiSortType.create_time,
        include_create_mi: true,
    })

    it('板名は完全一致・大小区別。null はすべて', () => {
        const kyou = make_kyou({ data_type: 'mi_create', typed_mi: make_mi({ board_name: '板A' }) })
        expect(does_kyou_match_query(kyou, Object.assign(base_query(), { mi_board_name: null }))).toBe(true)
        expect(does_kyou_match_query(kyou, Object.assign(base_query(), { mi_board_name: '板A' }))).toBe(true)
        expect(does_kyou_match_query(kyou, Object.assign(base_query(), { mi_board_name: '板a' }))).toBe(false)
    })

    it('mi_check_state は checked / uncheck のときだけ絞る', () => {
        const checked = make_kyou({ data_type: 'mi_create', typed_mi: make_mi({ is_checked: true }) })
        const unchecked = make_kyou({ data_type: 'mi_create', typed_mi: make_mi({ is_checked: false }) })
        expect(does_kyou_match_query(checked, Object.assign(base_query(), { mi_check_state: MiCheckState.checked }))).toBe(true)
        expect(does_kyou_match_query(unchecked, Object.assign(base_query(), { mi_check_state: MiCheckState.checked }))).toBe(false)
        expect(does_kyou_match_query(checked, Object.assign(base_query(), { mi_check_state: MiCheckState.uncheck }))).toBe(false)
        expect(does_kyou_match_query(unchecked, Object.assign(base_query(), { mi_check_state: MiCheckState.uncheck }))).toBe(true)
    })

    it('mi_check_state が空文字(未知の値)なら全通し', () => {
        const unknown_state = '' as MiCheckState
        const checked = make_kyou({ data_type: 'mi_create', typed_mi: make_mi({ is_checked: true }) })
        const unchecked = make_kyou({ data_type: 'mi_create', typed_mi: make_mi({ is_checked: false }) })
        expect(does_kyou_match_query(checked, Object.assign(base_query(), { mi_check_state: unknown_state }))).toBe(true)
        expect(does_kyou_match_query(unchecked, Object.assign(base_query(), { mi_check_state: unknown_state }))).toBe(true)
    })

    it('include_*_mi が全部falseなら0件', () => {
        const kyou = make_kyou({ data_type: 'mi_create', typed_mi: make_mi() })
        expect(does_kyou_match_query(kyou, Object.assign(base_query(), { include_create_mi: false }))).toBe(false)
    })

    it('カレンダーは「含める」指定のある射影の時刻で判定する', () => {
        const kyou = make_kyou({
            data_type: 'mi_create',
            typed_mi: make_mi({
                create_time: new Date('2026-08-01T00:00:00.000Z'),
                update_time: new Date('2026-08-01T00:00:00.000Z'),
                estimate_start_time: new Date('2026-09-15T00:00:00.000Z'),
            }),
        })
        // 9月の窓。create射影(8/1)だけを含めているので外れる
        const september = Object.assign(base_query(), {
            calendar_start_date: new Date('2026-09-01T00:00:00.000Z'),
            calendar_end_date: new Date('2026-09-30T00:00:00.000Z'),
        })
        expect(does_kyou_match_query(kyou, september)).toBe(false)
        // start射影(9/15)も含めれば通る
        expect(does_kyou_match_query(kyou, Object.assign(september, { include_start_mi: true }))).toBe(true)
    })

    it('mirekyou は typed_mirekyou を見る', () => {
        const kyou = make_kyou({
            data_type: 'mirekyou_create',
            typed_mi: null,
            typed_mirekyou: make_mi({ board_name: '板B' }),
        })
        expect(does_kyou_match_query(kyou, Object.assign(base_query(), { mi_board_name: '板B' }))).toBe(true)
    })
})

describe('apply_mi_projection', () => {
    const create_time = new Date('2026-08-01T09:00:00.000Z')
    const update_time = new Date('2026-08-05T21:00:00.000Z')
    const start_time = new Date('2026-08-10T10:00:00.000Z')

    it('ソート基準の時刻があれば、その時刻と接尾辞を入れる', () => {
        const kyou = make_kyou({
            data_type: 'mi_create',
            create_time: create_time,
            update_time: update_time,
            typed_mi: make_mi({ estimate_start_time: start_time }),
        })
        apply_mi_projection(kyou, MiSortType.estimate_start_time)
        expect(kyou.data_type).toBe('mi_start')
        expect(kyou.related_time).toEqual(start_time)
    })

    it('ソート基準の時刻が無ければ作成日時へフォールバックする(update_timeではない)', () => {
        const kyou = make_kyou({
            data_type: 'mi_create',
            create_time: create_time,
            update_time: update_time,
            typed_mi: make_mi({ estimate_start_time: null }),
        })
        apply_mi_projection(kyou, MiSortType.estimate_start_time)
        expect(kyou.data_type).toBe('mi_create')
        expect(kyou.related_time).toEqual(create_time)
    })

    it('create_timeソートでも作成日時を入れる(update_timeではない)', () => {
        const kyou = make_kyou({
            data_type: 'mi_create',
            create_time: create_time,
            update_time: update_time,
            typed_mi: make_mi(),
        })
        apply_mi_projection(kyou, MiSortType.create_time)
        expect(kyou.data_type).toBe('mi_create')
        expect(kyou.related_time).toEqual(create_time)
    })

    it('mirekyou は接頭辞を変える', () => {
        const kyou = make_kyou({
            data_type: 'mirekyou_create',
            create_time: create_time,
            typed_mirekyou: make_mi({ limit_time: start_time }),
        })
        apply_mi_projection(kyou, MiSortType.limit_time)
        expect(kyou.data_type).toBe('mirekyou_limit')
        expect(kyou.related_time).toEqual(start_time)
    })
})

describe('compare_kyou_for_query', () => {
    it('非miは related_time 降順', () => {
        const query = make_query({ for_mi: false })
        const newer = make_kyou({ id: 'a', related_time: new Date('2026-08-02T00:00:00.000Z') })
        const older = make_kyou({ id: 'b', related_time: new Date('2026-08-01T00:00:00.000Z') })
        expect(compare_kyou_for_query(newer, older, query)).toBeLessThan(0)
        expect(compare_kyou_for_query(older, newer, query)).toBeGreaterThan(0)
    })

    it('非miは同一秒内をIDの昇順で決める(サーバが秒に切り捨てて比べるため)', () => {
        const query = make_query({ for_mi: false })
        // ミリ秒では b のほうが新しいが、同一秒なのでIDで決まる
        const a = make_kyou({ id: 'a', related_time: new Date('2026-08-01T00:00:00.100Z') })
        const b = make_kyou({ id: 'b', related_time: new Date('2026-08-01T00:00:00.900Z') })
        expect(compare_kyou_for_query(a, b, query)).toBeLessThan(0)
    })

    it('miはソート基準の時刻の昇順', () => {
        const query = make_query({ for_mi: true, mi_sort_type: MiSortType.estimate_start_time })
        const early = make_kyou({ id: 'a', data_type: 'mi_start', related_time: new Date('2026-08-01T00:00:00.000Z') })
        const late = make_kyou({ id: 'b', data_type: 'mi_start', related_time: new Date('2026-08-02T00:00:00.000Z') })
        expect(compare_kyou_for_query(early, late, query)).toBeLessThan(0)
    })

    it('miはソート基準の時刻が無いものを末尾へ回す', () => {
        const query = make_query({ for_mi: true, mi_sort_type: MiSortType.estimate_start_time })
        // 未設定側は作成日時が入っていて時刻としては先だが、それでも後ろ
        const undated = make_kyou({ id: 'a', data_type: 'mi_create', related_time: new Date('2020-01-01T00:00:00.000Z') })
        const dated = make_kyou({ id: 'b', data_type: 'mi_start', related_time: new Date('2026-08-02T00:00:00.000Z') })
        expect(compare_kyou_for_query(dated, undated, query)).toBeLessThan(0)
        expect(compare_kyou_for_query(undated, dated, query)).toBeGreaterThan(0)
    })

    it('mi同着はIDの昇順', () => {
        const query = make_query({ for_mi: true, mi_sort_type: MiSortType.create_time })
        const same_time = new Date('2026-08-01T00:00:00.000Z')
        const a = make_kyou({ id: 'mi-a', data_type: 'mi_create', related_time: same_time })
        const b = make_kyou({ id: 'mi-b', data_type: 'mi_create', related_time: same_time })
        expect(compare_kyou_for_query(a, b, query)).toBeLessThan(0)
    })
})

describe('find_insert_index / insert_kyou_sorted', () => {
    const query = make_query({ for_mi: false })

    function make_list(): Array<Kyou> {
        return [
            make_kyou({ id: 'c', related_time: new Date('2026-08-03T00:00:00.000Z') }),
            make_kyou({ id: 'b', related_time: new Date('2026-08-02T00:00:00.000Z') }),
            make_kyou({ id: 'a', related_time: new Date('2026-08-01T00:00:00.000Z') }),
        ]
    }

    it('空リストへは先頭に入る', () => {
        expect(find_insert_index([], make_kyou(), query)).toBe(0)
    })

    it('先頭・中間・末尾を正しく求める', () => {
        const list = make_list()
        expect(find_insert_index(list, make_kyou({ id: 'z', related_time: new Date('2026-08-04T00:00:00.000Z') }), query)).toBe(0)
        expect(find_insert_index(list, make_kyou({ id: 'z', related_time: new Date('2026-08-02T12:00:00.000Z') }), query)).toBe(1)
        expect(find_insert_index(list, make_kyou({ id: 'z', related_time: new Date('2026-07-31T00:00:00.000Z') }), query)).toBe(3)
    })

    it('同一秒の中はIDの昇順で位置が決まる', () => {
        const same_time = new Date('2026-08-02T00:00:00.000Z')
        const list = [
            make_kyou({ id: 'b', related_time: same_time }),
            make_kyou({ id: 'd', related_time: same_time }),
        ]
        expect(find_insert_index(list, make_kyou({ id: 'a', related_time: same_time }), query)).toBe(0)
        expect(find_insert_index(list, make_kyou({ id: 'c', related_time: same_time }), query)).toBe(1)
        expect(find_insert_index(list, make_kyou({ id: 'e', related_time: same_time }), query)).toBe(2)
    })

    it('並び順を保って差し込む', () => {
        const list = make_list()
        const inserted = insert_kyou_sorted(list, make_kyou({ id: 'z', related_time: new Date('2026-08-02T12:00:00.000Z') }), query)
        expect(inserted).toBe(true)
        expect(list.map(kyou => kyou.id)).toEqual(['c', 'z', 'b', 'a'])
    })

    it('同じidが既にあれば何もしない', () => {
        const list = make_list()
        const inserted = insert_kyou_sorted(list, make_kyou({ id: 'b', related_time: new Date('2026-08-09T00:00:00.000Z') }), query)
        expect(inserted).toBe(false)
        expect(list.map(kyou => kyou.id)).toEqual(['c', 'b', 'a'])
    })

    it('配列は同じ参照のまま変更される(focused_kyous_listのエイリアスを切らない)', () => {
        const list = make_list()
        const same_reference = list
        insert_kyou_sorted(list, make_kyou({ id: 'z', related_time: new Date('2026-08-04T00:00:00.000Z') }), query)
        expect(list).toBe(same_reference)
        expect(list.length).toBe(4)
    })
})

describe('decide_local_insert', () => {
    it('判定できない条件の列は undecidable', () => {
        const decision = decide_local_insert(make_kyou(), make_query({ words: [] }))
        expect(decision.kind).toBe('undecidable')
    })

    it('for_mi列に非miのKyouは skip(再検索は不要)', () => {
        const query = make_query({ for_mi: true })
        expect(decide_local_insert(make_kyou({ data_type: 'kmemo' }), query).kind).toBe('skip')
    })

    it('非mi列のmi/timeisは undecidable', () => {
        const query = make_query({ for_mi: false })
        expect(decide_local_insert(make_kyou({ data_type: 'mi_create' }), query).kind).toBe('undecidable')
        expect(decide_local_insert(make_kyou({ data_type: 'timeis_start' }), query).kind).toBe('undecidable')
    })

    it('削除済みは skip', () => {
        expect(decide_local_insert(make_kyou({ is_deleted: true }), make_query()).kind).toBe('skip')
    })

    it('一致すれば insert で1行返す', () => {
        const decision = decide_local_insert(make_kyou(), make_query())
        expect(decision.kind).toBe('insert')
        if (decision.kind === 'insert') {
            expect(decision.rows.length).toBe(1)
        }
    })

    it('一致しなければ skip', () => {
        const decision = decide_local_insert(make_kyou({ rep_name: 'rep_a' }), make_query({ reps: ['rep_b'] }))
        expect(decision.kind).toBe('skip')
    })

    it('for_mi列では並び替え規則に合わせて related_time / data_type を書き換える', () => {
        const start_time = new Date('2026-08-10T10:00:00.000Z')
        const kyou = make_kyou({
            data_type: 'mi_create',
            typed_mi: make_mi({ estimate_start_time: start_time }),
        })
        const query = make_query({
            for_mi: true,
            mi_sort_type: MiSortType.estimate_start_time,
            mi_check_state: MiCheckState.all,
            include_create_mi: true,
            include_start_mi: true,
        })
        const decision = decide_local_insert(kyou, query)
        expect(decision.kind).toBe('insert')
        expect(kyou.data_type).toBe('mi_start')
        expect(kyou.related_time).toEqual(start_time)
    })
})
