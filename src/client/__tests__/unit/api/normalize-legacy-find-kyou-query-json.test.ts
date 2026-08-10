import { describe, test, expect } from 'vitest'
import {
    is_legacy_find_kyou_query_json,
    normalize_legacy_find_kyou_query_json,
} from '@/classes/api/find_query/normalize-legacy-find-kyou-query-json'

/**
 * 旧形式（use_*フラグ入り）FindKyouQuery JSON の正規化。
 * localStorage の保存クエリと SW にキャッシュされた古い application_config 応答は
 * クライアント側にしか残らないため、parse 境界のこの変換が唯一の防波堤。
 * 旧キーが1つでもインスタンスへ混入すると deep_equals のキー数比較が崩れて
 * サイドバーの「機械的な再emitを同値比較で捨てる」ガードが永久に効かなくなる。
 */

// 「キーごと消えていること」を表す番兵
const DELETED = Symbol('deleted')

type ConversionCase = {
    flag: string
    // フラグと一緒に与える値
    values: Record<string, unknown>
    // use_X=true のときの期待値（DELETED はキー自体が無いこと）
    when_true: Record<string, unknown>
    // use_X=false のときの期待値
    when_false: Record<string, unknown>
}

// 16フラグ×true/false の変換表。Go(find/find_query_legacy_json.go) と
// MCP(mcp/lib/constants.mjs) も同じ16キーを扱う（どれかが欠けると、そのフラグを送る
// 旧クライアントの保存クエリが移行されずに残る）。
// 配列系グループ: true=値維持 / false=null。nullable系グループ: true=値維持 / false=null。
// use_update_time は update_time ごと削除、use_mi_sort_type / use_mi_check_state /
// use_include_id / use_ids はキー削除のみで値には触らない。
const conversion_table: Array<ConversionCase> = [
    {
        flag: 'use_words',
        values: { words: ['a'], not_words: ['b'] },
        when_true: { words: ['a'], not_words: ['b'] },
        when_false: { words: null, not_words: null },
    },
    {
        flag: 'use_timeis',
        values: { timeis_words: ['w'], timeis_not_words: ['x'], timeis_tags: ['t'] },
        when_true: { timeis_words: ['w'], timeis_not_words: ['x'], timeis_tags: ['t'] },
        when_false: { timeis_words: null, timeis_not_words: null, timeis_tags: null },
    },
    {
        flag: 'use_timeis_tags',
        values: { timeis_tags: ['t'] },
        when_true: { timeis_tags: ['t'] },
        when_false: { timeis_tags: null },
    },
    {
        flag: 'use_tags',
        values: { tags: ['t1'] },
        when_true: { tags: ['t1'] },
        when_false: { tags: null },
    },
    {
        flag: 'use_reps',
        values: { reps: ['r1'] },
        when_true: { reps: ['r1'] },
        when_false: { reps: null },
    },
    {
        flag: 'use_rep_types',
        values: { rep_types: ['kmemo'] },
        when_true: { rep_types: ['kmemo'] },
        when_false: { rep_types: null },
    },
    {
        flag: 'use_map',
        values: { map_latitude: 35.1, map_longitude: 139.2, map_radius: 300 },
        when_true: { map_latitude: 35.1, map_longitude: 139.2, map_radius: 300 },
        when_false: { map_latitude: null, map_longitude: null, map_radius: null },
    },
    {
        flag: 'use_calendar',
        values: { calendar_start_date: '2026-01-01T00:00:00.000Z', calendar_end_date: '2026-01-31T00:00:00.000Z' },
        when_true: { calendar_start_date: '2026-01-01T00:00:00.000Z', calendar_end_date: '2026-01-31T00:00:00.000Z' },
        when_false: { calendar_start_date: null, calendar_end_date: null },
    },
    {
        flag: 'use_plaing',
        values: { plaing_time: '2026-01-15T12:00:00.000Z' },
        when_true: { plaing_time: '2026-01-15T12:00:00.000Z' },
        when_false: { plaing_time: null },
    },
    {
        flag: 'use_update_time',
        values: { update_time: '2026-01-20T09:00:00.000Z' },
        when_true: { update_time: DELETED },
        when_false: { update_time: DELETED },
    },
    {
        flag: 'use_period_of_time',
        values: { period_of_time_start_time_second: 3600, period_of_time_end_time_second: 7200, period_of_time_week_of_days: [1, 2, 3] },
        when_true: { period_of_time_start_time_second: 3600, period_of_time_end_time_second: 7200, period_of_time_week_of_days: [1, 2, 3] },
        when_false: { period_of_time_start_time_second: null, period_of_time_end_time_second: null, period_of_time_week_of_days: null },
    },
    {
        flag: 'use_mi_board_name',
        values: { mi_board_name: 'inbox' },
        when_true: { mi_board_name: 'inbox' },
        when_false: { mi_board_name: null },
    },
    {
        flag: 'use_mi_sort_type',
        values: { mi_sort_type: 'limit_time' },
        when_true: { mi_sort_type: 'limit_time' },
        when_false: { mi_sort_type: 'limit_time' },
    },
    {
        flag: 'use_mi_check_state',
        values: { mi_check_state: 'checked' },
        when_true: { mi_check_state: 'checked' },
        when_false: { mi_check_state: 'checked' },
    },
    {
        flag: 'use_include_id',
        values: {},
        when_true: {},
        when_false: {},
    },
    {
        // クライアントに ids フィールドは無いので値には触らない。
        // 列に載せるのは「旧JSONだと検知して新形式で書き戻す」ため
        flag: 'use_ids',
        values: { ids: ['id1'] },
        when_true: { ids: ['id1'] },
        when_false: { ids: ['id1'] },
    },
]

describe('normalize_legacy_find_kyou_query_json', () => {
    describe('16フラグ×true/false の変換表', () => {
        test.each(conversion_table)('$flag', ({ flag, values, when_true, when_false }) => {
            for (const [enabled, expected] of [[true, when_true], [false, when_false]] as const) {
                const input: Record<string, unknown> = { query_id: 'q1', ...values, [flag]: enabled }
                const { json, was_legacy } = normalize_legacy_find_kyou_query_json(input)

                expect(was_legacy).toBe(true)
                expect(json, `${flag}=${enabled}: フラグキーは削除されること`).not.toHaveProperty(flag)
                expect(json.query_id).toBe('q1')
                for (const [key, value] of Object.entries(expected)) {
                    if (value === DELETED) {
                        expect(json, `${flag}=${enabled}: ${key} はキーごと消えること`).not.toHaveProperty(key)
                    } else {
                        expect(json[key], `${flag}=${enabled}: ${key}`).toEqual(value)
                    }
                }
            }
        })

        test('全16フラグを同時に与えてもすべて削除される', () => {
            const input: Record<string, unknown> = { query_id: 'q-all' }
            for (const { flag, values } of conversion_table) {
                Object.assign(input, values)
                input[flag] = false
            }

            const { json, was_legacy } = normalize_legacy_find_kyou_query_json(input)

            expect(was_legacy).toBe(true)
            for (const { flag } of conversion_table) {
                expect(json).not.toHaveProperty(flag)
            }
            expect(is_legacy_find_kyou_query_json(json)).toBe(false)
        })
    })

    describe('TimeIs グループの複合ゲート', () => {
        test('use_timeis=false なら use_timeis_tags=true でも timeis_tags は null（旧ゲートは use_timeis && use_timeis_tags）', () => {
            const { json } = normalize_legacy_find_kyou_query_json({
                use_timeis: false,
                use_timeis_tags: true,
                timeis_words: ['w'],
                timeis_tags: ['t'],
            })

            expect(json.timeis_words).toBeNull()
            expect(json.timeis_not_words).toBeNull()
            expect(json.timeis_tags).toBeNull()
            expect(json).not.toHaveProperty('use_timeis')
            expect(json).not.toHaveProperty('use_timeis_tags')
        })

        test('use_timeis=true かつ use_timeis_tags=false なら timeis_tags だけ null', () => {
            const { json } = normalize_legacy_find_kyou_query_json({
                use_timeis: true,
                use_timeis_tags: false,
                timeis_words: ['w'],
                timeis_tags: ['t'],
            })

            expect(json.timeis_words).toEqual(['w'])
            expect(json.timeis_tags).toBeNull()
        })
    })

    describe('use_X=true の null/欠落値の物質化', () => {
        test('use_words=true で words / not_words が欠落していれば [] になる', () => {
            const { json } = normalize_legacy_find_kyou_query_json({ use_words: true, keywords: 'a' })
            expect(json.words).toEqual([])
            expect(json.not_words).toEqual([])
        })

        test('use_tags=true で tags が null なら [] になる（有効・チェック0個=0件の保存）', () => {
            const { json } = normalize_legacy_find_kyou_query_json({ use_tags: true, tags: null })
            expect(json.tags).toEqual([])
        })

        test('use_reps=true で reps が欠落していれば [] になる', () => {
            const { json } = normalize_legacy_find_kyou_query_json({ use_reps: true })
            expect(json.reps).toEqual([])
        })

        test('use_rep_types=true で rep_types が null なら [] になる', () => {
            const { json } = normalize_legacy_find_kyou_query_json({ use_rep_types: true, rep_types: null })
            expect(json.rep_types).toEqual([])
        })

        test('use_timeis=true で timeis_words / timeis_not_words が欠落していれば []（「任意のTimeIsに覆われたKyou」の保存）', () => {
            const { json } = normalize_legacy_find_kyou_query_json({ use_timeis: true })
            expect(json.timeis_words).toEqual([])
            expect(json.timeis_not_words).toEqual([])
        })

        test('use_timeis=true かつ use_timeis_tags=true で timeis_tags が null なら [] になる', () => {
            const { json } = normalize_legacy_find_kyou_query_json({ use_timeis: true, use_timeis_tags: true, timeis_tags: null })
            expect(json.timeis_tags).toEqual([])
        })

        test('use_period_of_time=true は week_of_days のみ [] を物質化し、start/end 秒には触らない', () => {
            const { json } = normalize_legacy_find_kyou_query_json({ use_period_of_time: true })
            expect(json.period_of_time_week_of_days).toEqual([])
            expect(json).not.toHaveProperty('period_of_time_start_time_second')
            expect(json).not.toHaveProperty('period_of_time_end_time_second')
        })

        test.each([
            { name: 'null', overrides: { mi_board_name: null } },
            { name: '欠落', overrides: {} },
        ])('use_mi_board_name=true で mi_board_name が $name なら "" になる（旧「空板名比較」の保存）', ({ overrides }) => {
            const { json } = normalize_legacy_find_kyou_query_json({ use_mi_board_name: true, ...overrides })
            expect(json.mi_board_name).toBe('')
        })

        test('nullable系グループ（use_map=true 等）は欠落値を物質化しない', () => {
            const { json } = normalize_legacy_find_kyou_query_json({ use_map: true, use_calendar: true, use_plaing: true })
            expect(json).not.toHaveProperty('map_latitude')
            expect(json).not.toHaveProperty('calendar_start_date')
            expect(json).not.toHaveProperty('plaing_time')
        })
    })

    describe('クライアント専用キーの保全', () => {
        test('keywords / query_id / mi_sort_type / mi_check_state 等の値はそのまま残る', () => {
            const { json } = normalize_legacy_find_kyou_query_json({
                query_id: 'col-1',
                keywords: 'foo -bar',
                timeis_keywords: 'baz',
                hide_tags: ['h'],
                devices_in_sidebar: ['laptop'],
                rep_types_in_sidebar: ['kmemo'],
                is_image_only: true,
                mi_sort_type: 'limit_time',
                mi_check_state: 'checked',
                use_words: false,
                use_mi_sort_type: false,
                use_mi_check_state: false,
            })

            expect(json.query_id).toBe('col-1')
            expect(json.keywords).toBe('foo -bar')
            expect(json.timeis_keywords).toBe('baz')
            expect(json.hide_tags).toEqual(['h'])
            expect(json.devices_in_sidebar).toEqual(['laptop'])
            expect(json.rep_types_in_sidebar).toEqual(['kmemo'])
            expect(json.is_image_only).toBe(true)
            expect(json.mi_sort_type).toBe('limit_time')
            expect(json.mi_check_state).toBe('checked')
        })
    })

    describe('update_time / use_update_time の削除', () => {
        test.each([true, false])('use_update_time=%s でも両方削除される', (enabled) => {
            const { json } = normalize_legacy_find_kyou_query_json({
                use_update_time: enabled,
                update_time: '2026-01-20T09:00:00.000Z',
            })
            expect(json).not.toHaveProperty('update_time')
            expect(json).not.toHaveProperty('use_update_time')
        })

        test('use_update_time が無くても他の旧フラグがあれば update_time は削除される', () => {
            const { json } = normalize_legacy_find_kyou_query_json({
                use_words: true,
                update_time: '2026-01-20T09:00:00.000Z',
            })
            expect(json).not.toHaveProperty('update_time')
        })
    })

    describe('冪等性', () => {
        test('2回通してもキー集合と値が同一になる', () => {
            const legacy = {
                query_id: 'q1',
                keywords: 'kw',
                use_words: true,
                words: ['kw'],
                use_tags: false,
                tags: ['t'],
                use_timeis: false,
                timeis_tags: ['tt'],
                use_update_time: true,
                update_time: '2026-01-01T00:00:00.000Z',
                use_mi_board_name: false,
                mi_board_name: 'inbox',
                use_include_id: false,
            }

            const first = normalize_legacy_find_kyou_query_json(legacy)
            expect(first.was_legacy).toBe(true)

            const second = normalize_legacy_find_kyou_query_json(first.json)
            expect(second.was_legacy).toBe(false)
            expect(Object.keys(second.json).sort()).toEqual(Object.keys(first.json).sort())
            expect(second.json).toEqual(first.json)
        })
    })

    describe('非レガシー入力の素通し', () => {
        test('was_legacy=false で同一参照が返る', () => {
            const modern = { query_id: 'q1', words: null, tags: [], reps: [], mi_board_name: null }

            const result = normalize_legacy_find_kyou_query_json(modern)

            expect(result.was_legacy).toBe(false)
            expect(result.json).toBe(modern)
        })

        test('is_legacy_find_kyou_query_json は use_* キーの有無だけで判定する', () => {
            expect(is_legacy_find_kyou_query_json({ words: null, tags: [] })).toBe(false)
            expect(is_legacy_find_kyou_query_json({ use_words: false })).toBe(true)
            expect(is_legacy_find_kyou_query_json({ use_include_id: true })).toBe(true)
        })
    })

    test('入力オブジェクト自体は破壊しない', () => {
        const legacy: Record<string, unknown> = { use_words: false, words: ['a'] }

        const { json } = normalize_legacy_find_kyou_query_json(legacy)

        expect(json).not.toBe(legacy)
        expect(legacy.use_words).toBe(false)
        expect(legacy.words).toEqual(['a'])
    })
})
