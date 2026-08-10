/**
 * サイドバー子クエリビューの「props同期ではemitしない」原則の検証。
 *
 * 列フォーカス切替でサイドバーが find_kyou_query を子へ同期したとき、
 * 子がそれを「ユーザーの条件変更」として emit すると、hot reload 既定ONの環境では
 * フォーカス切替のたびに実検索が発火し、飛行中の検索を abort して最初からやり直す
 * ループになる（2026-08-10 の rykv タブフリーズの発火源）。
 *
 * 固定する不変条件:
 * - TimeIsQuery: props同期は is_by_user=true の request_update_checked_timeis_tags を出さない。
 *   同期は pre_uncheck_all=true の「置き換え」で、列をまたいでチェックが累積しない
 * - MapQuery: props同期は request_update_area を出さない（radiusウォッチャ経由の間接発火も含む）
 * - CalendarQuery: 値の変わらない書き戻し（VDatePickerの正規化エコー）は request_update_dates を出さない
 * - ユーザーの実操作による emit は従来どおり通る
 */
import { describe, expect, test, vi } from 'vitest'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させないと class extends が undefined になる
import '@/classes/api/gkill-api'

import { nextTick, reactive } from 'vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { CheckState } from '@/pages/views/check-state'
import { useTimeIsQuery } from '@/classes/use-time-is-query'
import { useMapQuery } from '@/classes/use-map-query'
import { useCalendarQuery } from '@/classes/use-calendar-query'
import type { TimeIsQueryProps } from '@/pages/views/time-is-query-props'
import type { TimeIsQueryEmits } from '@/pages/views/time-is-query-emits'
import type { MapQueryProps } from '@/pages/views/map-query-props'
import type { MapQueryEmits } from '@/pages/views/map-query-emits'
import type { CalendarQueryProps } from '@/pages/views/calendar-query-props'
import type { CalendarQueryEmits } from '@/pages/views/calendar-query-emits'
import type { TagStructElementData } from '@/classes/datas/config/tag-struct-element-data'
import { makeTagStructElement } from '../../helpers/factory'

function collect_emits(): { emitted: Array<{ event: string, args: Array<unknown> }>, emits: (event: string, ...args: Array<unknown>) => void } {
    const emitted: Array<{ event: string, args: Array<unknown> }> = []
    const emits = (event: string, ...args: Array<unknown>) => {
        emitted.push({ event: event, args: args })
    }
    return { emitted, emits }
}

async function flush(times = 6): Promise<void> {
    for (let i = 0; i < times; i++) {
        await nextTick()
        await Promise.resolve()
    }
}

function make_tag_tree_config(): ApplicationConfig {
    const config = new ApplicationConfig()
    config.tag_struct = makeTagStructElement({
        name: 'root',
        key: '__root__',
        is_dir: true,
        children: [
            makeTagStructElement({ name: 'tagA', tag_name: 'tagA', key: 'tagA' }),
            makeTagStructElement({ name: 'tagB', tag_name: 'tagB', key: 'tagB' }),
        ],
    }) as unknown as TagStructElementData
    return config
}

function find_tag_node(config: ApplicationConfig, key: string): { is_checked: boolean } {
    const children = (config.tag_struct as unknown as { children: Array<{ key: string, is_checked: boolean }> }).children
    const node = children.find((child) => child.key === key)
    if (!node) {
        throw new Error(`tag node not found: ${key}`)
    }
    return node
}

describe('TimeIsQuery の props同期', () => {
    function createView() {
        const { emitted, emits } = collect_emits()
        const props = reactive({
            application_config: make_tag_tree_config(),
            gkill_api: {},
            find_kyou_query: new FindKyouQuery(),
            inited: true,
        }) as unknown as TimeIsQueryProps & { find_kyou_query: FindKyouQuery }
        const view = useTimeIsQuery({ props, emits: emits as unknown as TimeIsQueryEmits })
        return { view, props, emitted }
    }

    function by_user_emissions(emitted: Array<{ event: string, args: Array<unknown> }>): Array<{ event: string, args: Array<unknown> }> {
        return emitted.filter((emit) => emit.event === 'request_update_checked_timeis_tags' && emit.args[1] === true)
    }

    test('props同期は is_by_user=true の emit を出さない', async () => {
        const { props, emitted } = createView()
        const column_query = new FindKyouQuery()
        column_query.query_id = 'column-1'
        column_query.timeis_tags = ['tagA']
        props.find_kyou_query = column_query
        await flush()

        expect(by_user_emissions(emitted), '同期がユーザー編集として届くとフォーカス切替のたびに実検索が発火する').toHaveLength(0)
    })

    test('props同期は置き換え(pre_uncheck_all)で、列をまたいでチェックが累積しない', async () => {
        const { view, props } = createView()
        const column_a = new FindKyouQuery()
        column_a.query_id = 'column-a'
        column_a.timeis_tags = ['tagA']
        props.find_kyou_query = column_a
        await flush()
        expect(find_tag_node(view.cloned_application_config.value, 'tagA').is_checked).toBe(true)

        const column_b = new FindKyouQuery()
        column_b.query_id = 'column-b'
        column_b.timeis_tags = ['tagB']
        props.find_kyou_query = column_b
        await flush()

        expect(find_tag_node(view.cloned_application_config.value, 'tagB').is_checked).toBe(true)
        expect(find_tag_node(view.cloned_application_config.value, 'tagA').is_checked,
            '累積すると生成クエリが列クエリと一致せず、機械emitを値比較で吸収する安全網が破れる').toBe(false)
    })

    test('ユーザーのツリー操作(update_check_state)は is_by_user=true で emit される', async () => {
        const { view, emitted } = createView()
        view.foldable_struct.value = {
            get_selected_items: () => ['tagA'],
            update_check: vi.fn(),
        } as unknown as typeof view.foldable_struct.value

        await view.update_check_state(['tagA'], CheckState.checked)

        expect(by_user_emissions(emitted), 'ユーザー操作の emit まで殺してはいけない').toHaveLength(1)
    })
})

describe('MapQuery の props同期', () => {
    function createView() {
        const { emitted, emits } = collect_emits()
        const props = reactive({
            application_config: { google_map_api_key: '' },
            gkill_api: { get_google_map_api_key: () => '' },
            find_kyou_query: new FindKyouQuery(),
            inited: true,
        }) as unknown as MapQueryProps & { find_kyou_query: FindKyouQuery }
        const view = useMapQuery({ props, emits: emits as unknown as MapQueryEmits })
        return { view, props, emitted }
    }

    function area_emissions(emitted: Array<{ event: string, args: Array<unknown> }>): Array<{ event: string, args: Array<unknown> }> {
        return emitted.filter((emit) => emit.event === 'request_update_area')
    }

    test('props同期は request_update_area を出さない(radiusウォッチャ経由も含む)', async () => {
        const { view, props, emitted } = createView()
        const column_query = new FindKyouQuery()
        column_query.query_id = 'column-1'
        column_query.map_latitude = 35
        column_query.map_longitude = 139
        column_query.map_radius = 700
        props.find_kyou_query = column_query
        await flush()

        expect(view.radius.value, '同期そのものは行われる').toBe(700)
        expect(area_emissions(emitted), '同期がemitになるとフォーカス切替のたびに実検索が発火する').toHaveLength(0)
    })

    test('ユーザーのスライダー操作は emit され、続く同期では再emitしない', async () => {
        const { view, props, emitted } = createView()
        const column_query = new FindKyouQuery()
        column_query.query_id = 'column-1'
        column_query.map_latitude = 35
        column_query.map_longitude = 139
        column_query.map_radius = 700
        props.find_kyou_query = column_query
        await flush()

        view.radius.value = 999
        await flush()
        expect(area_emissions(emitted)).toHaveLength(1)
        expect(area_emissions(emitted)[0].args).toEqual([35, 139, 999])

        const other_column = new FindKyouQuery()
        other_column.query_id = 'column-2'
        other_column.map_latitude = 36
        other_column.map_longitude = 140
        other_column.map_radius = 1200
        props.find_kyou_query = other_column
        await flush()

        expect(view.radius.value).toBe(1200)
        expect(area_emissions(emitted), '別列への同期が再emitになってはいけない').toHaveLength(1)
    })
})

describe('CalendarQuery の props同期エコー', () => {
    function createView() {
        const { emitted, emits } = collect_emits()
        const props = reactive({
            application_config: {},
            gkill_api: {},
            find_kyou_query: new FindKyouQuery(),
            inited: true,
        }) as unknown as CalendarQueryProps & { find_kyou_query: FindKyouQuery }
        const view = useCalendarQuery({ props, emits: emits as unknown as CalendarQueryEmits })
        return { view, props, emitted }
    }

    function date_emissions(emitted: Array<{ event: string, args: Array<unknown> }>): Array<{ event: string, args: Array<unknown> }> {
        return emitted.filter((emit) => emit.event === 'request_update_dates')
    }

    test('値の変わらない書き戻しは emit しない', async () => {
        const { view, props, emitted } = createView()

        // 未選択のまま空配列が書き戻されても emit しない
        view.clicked_date([])
        expect(date_emissions(emitted)).toHaveLength(0)

        const column_query = new FindKyouQuery()
        column_query.query_id = 'column-1'
        column_query.calendar_start_date = new Date(2025, 2, 10)
        column_query.calendar_end_date = new Date(2025, 2, 12, 23, 59, 59, 999)
        props.find_kyou_query = column_query
        await flush()
        expect(view.dates.value).toHaveLength(2)

        // VDatePickerの正規化エコー(同じ日付)は emit しない
        view.clicked_date([new Date(2025, 2, 10), new Date(2025, 2, 12)])
        expect(date_emissions(emitted), '同期のエコーがemitになるとフォーカス切替のたびに実検索が発火する').toHaveLength(0)
    })

    test('ユーザーの実変更とクリアは emit される', async () => {
        const { view, props, emitted } = createView()
        const column_query = new FindKyouQuery()
        column_query.query_id = 'column-1'
        column_query.calendar_start_date = new Date(2025, 2, 10)
        column_query.calendar_end_date = new Date(2025, 2, 12, 23, 59, 59, 999)
        props.find_kyou_query = column_query
        await flush()

        view.clicked_date([new Date(2025, 2, 10), new Date(2025, 2, 13)])
        expect(date_emissions(emitted)).toHaveLength(1)
        expect((date_emissions(emitted)[0].args[1] as Date).getDate(), '終端はその日の終わりに丸められる').toBe(13)

        view.clicked_date([])
        expect(date_emissions(emitted)).toHaveLength(2)
        expect(date_emissions(emitted)[1].args).toEqual([null, null])
    })
})
