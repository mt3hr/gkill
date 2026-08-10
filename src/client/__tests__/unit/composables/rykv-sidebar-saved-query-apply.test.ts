/**
 * サイドバーの「保存済み検索条件の適用」の検証（rykv側）。
 *
 * use-rykv-query-editor-side-bar.ts と use-mi-query-editor-sidebar.ts はコピー由来なので、
 * 片側を直したら mi-sidebar-saved-query-apply.test.ts も見ること。
 * ただし節の構成はもう対称ではない ―― mi 側からは状況(TimeIs)・時間帯・場所を外してある。
 *
 * 検証する不変条件:
 * - 適用しても query_id は列側を維持する（保存条件由来の query_id を列へ持ち込むと
 *   「列×検索」の不変条件が崩れ、検索結果の誤配送が再発する）
 * - 適用は updated_query の emit 1回だけ（手編集と同じ扱い。検索を実行するかは
 *   ホットリロード設定に従って親が決める）
 * - emit されるのは保存側の clone（emit 先で書き換えても保存アイテムが汚れない）
 * - saved_find_querys は設定が無ければ空（FAB の v-if 非表示条件の実体）
 */
import { describe, expect, test } from 'vitest'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させないと class extends が undefined になる
import '@/classes/api/gkill-api'

import { nextTick, reactive } from 'vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { SavedFindQueryConfig } from '@/classes/datas/config/saved-find-query-config'
import { useRykvQueryEditorSideBar } from '@/classes/use-rykv-query-editor-side-bar'
import type { RykvQueryEditorSidebarProps } from '@/pages/views/rykv-query-editor-sidebar-props'
import type { RykvQueryEditorSidebarEmits } from '@/pages/views/rykv-query-editor-sidebar-emits'

function make_saved_config_json(): Record<string, unknown> {
    const saved_query = new FindKyouQuery()
    saved_query.query_id = 'saved-query-id'
    // words 非null = キーワードフィルタ有効（use_* フラグは全廃済み）
    saved_query.keywords = '写真'
    saved_query.words = ['写真']
    saved_query.not_words = []
    saved_query.tags = ['旅行']
    const config = new SavedFindQueryConfig()
    config.saved_rykv_find_kyou_querys = [{ id: 'item-1', title: '旅行の写真', find_kyou_query: saved_query }]
    return config.to_json()
}

function createView(saved_json?: Record<string, unknown>) {
    const application_config = new ApplicationConfig()
    if (saved_json) {
        application_config.saved_find_query_json_data = saved_json
    }
    const emitted: Array<{ event: string, payload: unknown }> = []
    const emits = ((event: string, ...args: Array<unknown>) => {
        emitted.push({ event: event, payload: args[0] })
    }) as unknown as RykvQueryEditorSidebarEmits
    const props = reactive({
        application_config: application_config,
        gkill_api: { generate_uuid: () => 'generated-uuid' },
        find_kyou_query: new FindKyouQuery(),
        inited: false,
        app_title_bar_height: 50,
        app_content_height: 800,
        app_content_width: 1200,
    }) as unknown as RykvQueryEditorSidebarProps & { find_kyou_query: FindKyouQuery }
    const view = useRykvQueryEditorSideBar({ props, emits })
    return { view, props, emitted }
}

describe('rykvサイドバーの保存済み検索条件適用', () => {
    test('適用は updated_query を1回だけ emit し、query_id は列側を維持する', async () => {
        const { view, props, emitted } = createView(make_saved_config_json())
        const column_query = new FindKyouQuery()
        column_query.query_id = 'column-1'
        props.find_kyou_query = column_query
        await nextTick()

        const item = view.saved_find_querys.value[0]
        await view.apply_saved_query(item)

        const updated = emitted.filter((emit) => emit.event === 'updated_query')
        expect(updated, 'updated_query は1回だけ').toHaveLength(1)
        const applied = updated[0].payload as FindKyouQuery
        expect(applied.query_id, '保存条件の query_id を列へ持ち込んではいけない').toBe('column-1')
        expect(applied.keywords).toBe('写真')
        expect(applied.words, '非null=キーワードフィルタ有効').toEqual(['写真'])
        expect(applied.tags).toEqual(['旅行'])
    })

    test('emit されるのは保存側の clone（書き換えても保存アイテムが汚れない）', async () => {
        const { view, props, emitted } = createView(make_saved_config_json())
        const column_query = new FindKyouQuery()
        column_query.query_id = 'column-1'
        props.find_kyou_query = column_query
        await nextTick()

        const item = view.saved_find_querys.value[0]
        await view.apply_saved_query(item)

        const applied = emitted[0].payload as FindKyouQuery
        expect(applied, '同一インスタンスだと列側の編集が保存条件へ逆流する').not.toBe(item.find_kyou_query)
        applied.words?.push('汚染')
        applied.tags?.push('汚染')
        expect(item.find_kyou_query.words).toEqual(['写真'])
        expect(item.find_kyou_query.tags).toEqual(['旅行'])
    })

    test('saved_find_querys は設定が無ければ空（FAB非表示条件）、あれば件数分', () => {
        const empty = createView()
        expect(empty.view.saved_find_querys.value).toEqual([])

        const filled = createView(make_saved_config_json())
        expect(filled.view.saved_find_querys.value).toHaveLength(1)
        expect(filled.view.saved_find_querys.value[0].title).toBe('旅行の写真')
    })

    test('rykvサイドバーはタスク側の保存条件を表示しない', () => {
        const config = new SavedFindQueryConfig()
        config.saved_mi_find_kyou_querys = [{ id: 'm1', title: 'タスク側', find_kyou_query: new FindKyouQuery() }]
        const { view } = createView(config.to_json())
        expect(view.saved_find_querys.value).toEqual([])
    })
})
