/**
 * miサイドバーの節ごとの inited フラグの検証。
 *
 * このフラグは子へ :inited prop として降り、子（use-tag-query.ts /
 * use-calendar-query.ts）は「初回同期か再同期か」の判定に使う。立たないと
 * props 同期のたびにチェックが列をまたいで累積する。
 *
 * かつてはこれらの AND を親への '@inited' イベントにして画面の初期化を起動
 * していたが、その集約は「設定が来た」を表していたわけではなく、
 * 「immediateの付いていない application_config watch から emit する子がいる」
 * という偶然に乗っていただけだった（miでは実質 CalendarQuery 1つが律速し、
 * しかもその節は application_config のフィールドを1つも読まない）。
 * そのため節を1つ画面から外すだけで画面ごとスピナーで固まっていた。
 * いまは use-mi-view.ts が application_config.is_loaded を直接 watch して
 * 初期化するので、集約が復活していないこともここで固定する。
 *
 * 状況(TimeIs)・時間帯・場所の3節は mi から外した。
 * use-mi-find-query-editor-view.ts の inited 集約は別コンポーネント
 * （検索条件エディタダイアログ）のもので、画面の初期化とは無関係なので残っている。
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
import { useMiQueryEditorSidebar } from '@/classes/use-mi-query-editor-sidebar'
import { useMiFindQueryEditorView } from '@/classes/use-mi-find-query-editor-view'
import type { MiQueryEditorSidebarProps } from '@/pages/views/mi-query-editor-sidebar-props'
import type { MiQueryEditorSidebarEmits } from '@/pages/views/mi-query-editor-sidebar-emits'
import type { MiFindQueryEditorViewProps } from '@/pages/views/mi-find-query-editor-view-props'
import type { MiFindQueryEditorViewEmits } from '@/pages/views/mi-find-query-editor-view-emits'

function make_props() {
    return {
        application_config: new ApplicationConfig(),
        gkill_api: { generate_uuid: () => 'generated-uuid' },
        find_kyou_query: new FindKyouQuery(),
        app_title_bar_height: 50,
        app_content_height: 800,
        app_content_width: 1200,
    }
}

function collect_emits(emitted: Array<string>) {
    return ((event: string) => { emitted.push(event) }) as unknown as never
}

describe('miサイドバーの節ごとの inited フラグ', () => {
    test('各節の @inited が対応するフラグを立てる', async () => {
        const emitted: Array<string> = []
        const props = reactive(make_props()) as unknown as MiQueryEditorSidebarProps
        const view = useMiQueryEditorSidebar({
            props: props,
            emits: collect_emits(emitted) as unknown as MiQueryEditorSidebarEmits,
        })
        await nextTick()

        // キーワードとヘッダは最初から true。残りは子の @inited を待つ
        expect(view.inited_keyword_query_for_query_sidebar.value).toBe(true)
        expect(view.inited_sidebar_header_for_query_sidebar.value).toBe(true)
        expect(view.inited_tag_query_for_query_sidebar.value, '何も来ていないのにタグ節が立っている').toBe(false)
        expect(view.inited_calendar_query_for_query_sidebar.value).toBe(false)
        expect(view.inited_check_state_query_for_query_sidebar.value).toBe(false)
        expect(view.inited_sort_query_for_query_sidebar.value).toBe(false)
        expect(view.inited_board_query_for_query_sidebar.value).toBe(false)

        view.onInitedTag()
        view.onInitedCalendar()
        view.onInitedCheckState()
        view.onInitedSort()
        view.onInitedBoard()
        await nextTick()

        // このフラグは子へ :inited prop として降り、子は「初回同期か再同期か」の
        // 判定に使う。立たないと props 同期のたびにチェックが累積する
        expect(view.inited_tag_query_for_query_sidebar.value, 'タグ節のフラグが立たない').toBe(true)
        expect(view.inited_calendar_query_for_query_sidebar.value).toBe(true)
        expect(view.inited_check_state_query_for_query_sidebar.value).toBe(true)
        expect(view.inited_sort_query_for_query_sidebar.value).toBe(true)
        expect(view.inited_board_query_for_query_sidebar.value).toBe(true)
    })

    test('サイドバーは画面の初期化トリガを持たない', async () => {
        // 画面の初期化は use-mi-view.ts が application_config.is_loaded を直接
        // watch して起こす。以前はこのサイドバーの集約 inited を @inited として
        // 上げて起動していたが、「設定が来た」を表していたのは
        // 「immediateの付いていない application_config watch から emit する子がいる」
        // という偶然で、実質 CalendarQuery 1つが律速していた。
        // 集約が復活すると、その節を画面から外したときに画面ごと固まる形へ戻る
        const emitted: Array<string> = []
        const props = reactive(make_props()) as unknown as MiQueryEditorSidebarProps
        const view = useMiQueryEditorSidebar({
            props: props,
            emits: collect_emits(emitted) as unknown as MiQueryEditorSidebarEmits,
        }) as unknown as Record<string, unknown>
        expect(view.inited, '集約 inited が復活している').toBeUndefined()

        const typed_view = view as unknown as ReturnType<typeof useMiQueryEditorSidebar>
        typed_view.onInitedTag()
        typed_view.onInitedCalendar()
        typed_view.onInitedCheckState()
        typed_view.onInitedSort()
        typed_view.onInitedBoard()
        await nextTick()
        await nextTick()
        expect(emitted, "'@inited' を emit している").not.toContain('inited')
    })

    test('状況・時間帯・場所のハンドラはもう公開されていない', () => {
        const props = reactive(make_props()) as unknown as MiQueryEditorSidebarProps
        const view = useMiQueryEditorSidebar({
            props: props,
            emits: collect_emits([]) as unknown as MiQueryEditorSidebarEmits,
        }) as unknown as Record<string, unknown>
        for (const removed of ['onInitedTimeis', 'onInitedMap', 'timeis_query', 'map_query', 'period_of_time_query']) {
            expect(view[removed], `${removed} が残っている`).toBeUndefined()
        }
    })
})

describe('mi検索条件エディタの inited 集約', () => {
    test('画面に残っている節の @inited が揃えば loading が晴れる', async () => {
        const emitted: Array<string> = []
        const props = reactive(make_props()) as unknown as MiFindQueryEditorViewProps
        const view = useMiFindQueryEditorView({
            props: props,
            emits: collect_emits(emitted) as unknown as MiFindQueryEditorViewEmits,
        })
        await nextTick()

        expect(view.loading.value).toBe(true)

        view.onInitedTag()
        view.onInitedCheckState()
        view.onInitedSort()
        await nextTick()
        expect(view.inited.value, '画面に残る節が揃っても inited が立たない').toBe(true)

        // loading の解除は inited の watcher → nextTick 2段で走る
        await nextTick()
        await nextTick()
        await nextTick()
        expect(view.loading.value, 'inited が立ったのに loading が晴れない（保存ボタンが出ない）').toBe(false)
    })
})
