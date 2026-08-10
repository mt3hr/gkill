/**
 * miサイドバーの inited 集約の検証。
 *
 * inited が true にならないと '@inited' が飛ばず、use-mi-view.ts の onSidebarInited() が
 * init() を呼べないため、mi画面が is_loading=true のままスピナーで固まる（見た目ではなく画面ハング）。
 * 画面から節を消したときに、その節の inited_* フラグを computed から外し忘れると
 * 誰も true にしないフラグが残って必ずこうなるので、フラグ集合をここで固定する。
 *
 * 状況(TimeIs)・時間帯・場所の3節は mi から外したので、そのフラグは集約に含めない。
 * 同じ理由で use-mi-find-query-editor-view.ts 側も検査する。
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

describe('miサイドバーの inited 集約', () => {
    test('画面に残っている節の @inited が揃えば inited が true になる', async () => {
        const emitted: Array<string> = []
        const props = reactive(make_props()) as unknown as MiQueryEditorSidebarProps
        const view = useMiQueryEditorSidebar({
            props: props,
            emits: collect_emits(emitted) as unknown as MiQueryEditorSidebarEmits,
        })
        await nextTick()

        // キーワードとヘッダは最初から true。残りは子の @inited を待つ
        expect(view.inited.value, '何も来ていないのに inited が立っている').toBe(false)

        view.onInitedTag()
        view.onInitedCalendar()
        view.onInitedCheckState()
        view.onInitedSort()
        view.onInitedBoard()
        await nextTick()

        expect(view.inited.value, '画面に残る節が揃っても inited が立たない（消した節のフラグが残っている疑い）').toBe(true)
        await nextTick()
        expect(emitted, "inited が立ったのに '@inited' を emit していない").toContain('inited')
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
