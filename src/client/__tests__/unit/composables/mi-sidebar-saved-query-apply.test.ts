/**
 * サイドバーの「保存済み検索条件の適用」の検証（mi側）。
 *
 * use-mi-query-editor-sidebar.ts と use-rykv-query-editor-side-bar.ts はコピー由来なので、
 * 片側を直したら rykv-sidebar-saved-query-apply.test.ts も見ること。
 * ただし節の構成はもう対称ではない ―― mi 側からは状況(TimeIs)・時間帯・場所を外してあり、
 * それらのフィールドは保存も復元もされない（rykv 側には残っている）。
 *
 * rykv側と同じ不変条件に加えて、mi固有の仕様を固定する:
 * - 板名は全クリア（emits_default_query）と違って現在の列の板を保持せず、
 *   保存された条件の板名（mi_board_name。null=「すべて」）がそのまま勝つ
 */
import { describe, expect, test, vi } from 'vitest'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させないと class extends が undefined になる
import '@/classes/api/gkill-api'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

import { nextTick, reactive } from 'vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import { SavedFindQueryConfig } from '@/classes/datas/config/saved-find-query-config'
import { useMiQueryEditorSidebar } from '@/classes/use-mi-query-editor-sidebar'
import type { MiQueryEditorSidebarProps } from '@/pages/views/mi-query-editor-sidebar-props'
import type { MiQueryEditorSidebarEmits } from '@/pages/views/mi-query-editor-sidebar-emits'

function make_saved_config_json(): Record<string, unknown> {
    const saved_query = new FindKyouQuery()
    saved_query.query_id = 'saved-query-id'
    saved_query.for_mi = true
    // words 非null = キーワードフィルタ有効、mi_board_name 非null = 板絞り込みあり
    saved_query.keywords = '買い物'
    saved_query.words = ['買い物']
    saved_query.not_words = []
    saved_query.mi_board_name = '保存された板'
    const config = new SavedFindQueryConfig()
    config.saved_mi_find_kyou_querys = [{ id: 'item-1', title: '今週のタスク', find_kyou_query: saved_query }]
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
    }) as unknown as MiQueryEditorSidebarEmits
    const props = reactive({
        application_config: application_config,
        gkill_api: { generate_uuid: () => 'generated-uuid' },
        find_kyou_query: new FindKyouQuery(),
        app_title_bar_height: 50,
        app_content_height: 800,
        app_content_width: 1200,
    }) as unknown as MiQueryEditorSidebarProps & { find_kyou_query: FindKyouQuery }
    const view = useMiQueryEditorSidebar({ props, emits })
    return { view, props, emitted }
}

describe('miサイドバーの保存済み検索条件適用', () => {
    test('適用は updated_query を1回だけ emit し、query_id は列側を維持する', async () => {
        const { view, props, emitted } = createView(make_saved_config_json())
        const column_query = new FindKyouQuery()
        column_query.query_id = 'mi-column-1'
        props.find_kyou_query = column_query
        await nextTick()

        const item = view.saved_find_querys.value[0]
        await view.apply_saved_query(item)

        const updated = emitted.filter((emit) => emit.event === 'updated_query')
        expect(updated, 'updated_query は1回だけ').toHaveLength(1)
        const applied = updated[0].payload as FindKyouQuery
        expect(applied.query_id, '保存条件の query_id を列へ持ち込んではいけない').toBe('mi-column-1')
        expect(applied.keywords).toBe('買い物')
        expect(applied.words, '非null=キーワードフィルタ有効').toEqual(['買い物'])
    })

    test('板名は保存された条件が勝つ（全クリアの板名保持とは違う）', async () => {
        const { view, props, emitted } = createView(make_saved_config_json())
        const column_query = new FindKyouQuery()
        column_query.query_id = 'mi-column-1'
        column_query.mi_board_name = '現在の板'
        props.find_kyou_query = column_query
        await nextTick()

        await view.apply_saved_query(view.saved_find_querys.value[0])

        const applied = emitted[0].payload as FindKyouQuery
        expect(applied.mi_board_name, '現在の列の板を保持してはいけない').toBe('保存された板')
    })

    test('保存条件が「すべて」(mi_board_name=null)でも板列に適用でき、nullが勝つ', async () => {
        const saved_query = new FindKyouQuery()
        saved_query.query_id = 'saved-all'
        saved_query.for_mi = true
        // mi_board_name はコンストラクタ既定の null（=「すべて」）のまま
        const config = new SavedFindQueryConfig()
        config.saved_mi_find_kyou_querys = [{ id: 'item-all', title: 'すべての板', find_kyou_query: saved_query }]

        const { view, props, emitted } = createView(config.to_json())
        const column_query = new FindKyouQuery()
        column_query.query_id = 'mi-column-1'
        column_query.mi_board_name = '現在の板'
        props.find_kyou_query = column_query
        await nextTick()

        await view.apply_saved_query(view.saved_find_querys.value[0])

        const applied = emitted[0].payload as FindKyouQuery
        expect(applied.mi_board_name, 'null=「すべて」も条件の一部として勝つ').toBeNull()
    })

    test('emit されるのは保存側の clone（書き換えても保存アイテムが汚れない）', async () => {
        const { view, props, emitted } = createView(make_saved_config_json())
        const column_query = new FindKyouQuery()
        column_query.query_id = 'mi-column-1'
        props.find_kyou_query = column_query
        await nextTick()

        const item = view.saved_find_querys.value[0]
        await view.apply_saved_query(item)

        const applied = emitted[0].payload as FindKyouQuery
        expect(applied).not.toBe(item.find_kyou_query)
        applied.words?.push('汚染')
        expect(item.find_kyou_query.words).toEqual(['買い物'])
    })

    test('saved_find_querys は設定が無ければ空（FAB非表示条件）、あれば件数分', () => {
        const empty = createView()
        expect(empty.view.saved_find_querys.value).toEqual([])

        const filled = createView(make_saved_config_json())
        expect(filled.view.saved_find_querys.value).toHaveLength(1)
        expect(filled.view.saved_find_querys.value[0].title).toBe('今週のタスク')
    })

    test('miサイドバーはライフログ側の保存条件を表示しない', () => {
        const config = new SavedFindQueryConfig()
        config.saved_rykv_find_kyou_querys = [{ id: 'r1', title: 'ライフログ側', find_kyou_query: new FindKyouQuery() }]
        const { view } = createView(config.to_json())
        expect(view.saved_find_querys.value).toEqual([])
    })
})
