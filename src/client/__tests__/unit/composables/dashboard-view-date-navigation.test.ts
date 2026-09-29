/**
 * useDashboardView の表示日の移動（地図の日付ピッカー → go_date）の検証。
 *
 * ダッシュボードの Dnote・Mi リスト・地図はどれも selected_date（表示日）に従う。
 * 地図の日付表示をクリックして開くピッカーは requested_change_map_date で選んだ日を親へ返し、
 * ダッシュボードでは go_date が表示日ごと移す（rykv は地図だけを切り替えるのと対照的）。
 *
 * go_date の中身を消しても型検査は通り、「ピッカーで日を選んでも何も起きない」がエラー無しで起きる。
 * 表示日の変化で取り直す watch(selected_date) は dashboard-view.vue 側にあるので、
 * ここではコンポーザブルの表示日と、テンプレートの配線（ソース走査）を見る。
 *
 * 再読込まわりは dashboard-view-reload.test.ts、ApplicationConfig の取得はページ側。
 */
import { describe, expect, it, vi, afterEach, beforeEach } from 'vitest'
import { existsSync, readFileSync } from 'node:fs'
import { dirname, join } from 'node:path'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))
// router は全ページを引き込むので、この画面が使う replace だけ差し替える
vi.mock('@/router', () => ({
    default: { replace: vi.fn(), push: vi.fn() },
}))
vi.mock('@/classes/delete-gkill-cache', () => ({
    default: vi.fn().mockResolvedValue(undefined),
    delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
    delete_gkill_all_tag_names_cache: vi.fn().mockResolvedValue(undefined),
    delete_gkill_attached_datas_cache: vi.fn().mockResolvedValue(undefined),
}))
vi.mock('@/classes/use-dialog-history-stack', () => ({
    reset_dialog_history: vi.fn().mockResolvedValue(undefined),
}))
vi.mock('@/classes/use-scoped-enter-for-kftl', () => ({ useScopedEnterForKFTL: vi.fn() }))
vi.mock('@/classes/use-scoped-ctrl-v-for-clipboard', () => ({ useScopedCtrlVForClipboard: vi.fn() }))
vi.mock('@/classes/kyou-reload', () => ({
    new_reload_batch: vi.fn(() => 0),
    refresh_kyou: vi.fn().mockResolvedValue(null),
    refresh_kyou_in_list: vi.fn().mockResolvedValue(undefined),
    build_mi_reload_query: vi.fn((query: unknown) => query),
}))

import { createApp, defineComponent, h, reactive } from 'vue'
import { useDashboardView } from '@/classes/use-dashboard-view'
import { GkillAPI } from '@/classes/api/gkill-api'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { DashboardViewProps } from '@/pages/views/dashboard-view-props'
import type { DashboardViewEmits } from '@/pages/views/dashboard-view-emits'

function make_fake_api() {
    return {
        get_session_id: vi.fn(() => 'test-session'),
        generate_uuid: vi.fn(() => 'generated-uuid'),
        set_use_dark_theme: vi.fn(),
        set_saved_application_config: vi.fn(),
        get_kyous: vi.fn().mockResolvedValue({ kyous: [], messages: [], errors: [] }),
        get_kyou: vi.fn().mockResolvedValue({ kyou_histories: [], messages: [], errors: [] }),
        get_all_tag_names: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
        get_mi_board_list: vi.fn().mockResolvedValue({ boards: [], messages: [], errors: [] }),
    }
}

function make_view_application_config(): ApplicationConfig {
    const application_config = new ApplicationConfig()
    application_config.is_loaded = true
    return application_config
}

let mounted_apps = new Array<ReturnType<typeof createApp>>()

function mount_view() {
    let view: ReturnType<typeof useDashboardView> | null = null
    const props = reactive({
        gkill_api: make_fake_api() as unknown as GkillAPI,
        application_config: make_view_application_config(),
        app_title_bar_height: 50,
        app_content_height: 900,
        app_content_width: 1200,
        application_config_load_failed: false,
        is_hosted_in_dialog: false,
    }) as unknown as DashboardViewProps
    const emits = (() => { }) as unknown as DashboardViewEmits
    const Host = defineComponent({
        setup() {
            view = useDashboardView({ props, emits })
            return () => h('div')
        },
    })
    const app = createApp(Host)
    app.mount(document.createElement('div'))
    mounted_apps.push(app)
    return { app: app, view: view! }
}

beforeEach(() => {
    vi.spyOn(GkillAPI, 'get_instance').mockReturnValue(make_fake_api() as unknown as GkillAPI)
})

afterEach(() => {
    for (const app of mounted_apps) {
        app.unmount()
    }
    mounted_apps = []
    vi.restoreAllMocks()
})

describe('go_date（地図の日付ピッカーで選んだ日）', () => {
    it('表示日をその日の 0 時へ移す', () => {
        const { view } = mount_view()

        view.go_date(new Date(2016, 4, 15, 13, 45, 30))

        // 時刻付きで選んでも表示日は 0 時に丸める（前日・翌日の移動と同じ形）
        expect(view.selected_date.value.getTime(), '表示日が移っていない').toBe(new Date(2016, 4, 15, 0, 0, 0).getTime())
    })

    it('Dnote・Mi リスト・地図が従う日の両端と日付表示が、選んだ日になる', () => {
        const { view } = mount_view()

        view.go_date(new Date(2016, 4, 15, 13, 45, 30))

        expect(view.target_date_start.value.getTime()).toBe(new Date(2016, 4, 15, 0, 0, 0).getTime())
        expect(view.target_date_end.value.getTime()).toBe(new Date(2016, 4, 15, 23, 59, 59, 999).getTime())
        // i18n は key をそのまま返すモック。2016-05-15 は日曜
        expect(view.date_label.value).toBe('2016/5/15(SUNDAY_TITLE)')
        // ヘッダの日付ピッカー（表示値は selected_date）も同じ日を指す
        expect(view.date_picker_model.value.getTime()).toBe(new Date(2016, 4, 15, 0, 0, 0).getTime())
    })

    it('go_today で今日へ戻れる（ピッカーで飛んだ先に固定されない）', () => {
        const { view } = mount_view()
        view.go_date(new Date(2016, 4, 15))

        view.go_today()

        const today = new Date()
        expect(view.selected_date.value.getTime()).toBe(new Date(today.getFullYear(), today.getMonth(), today.getDate()).getTime())
    })
})

// import.meta.url は vitest の変換後は file スキームにならないので使えない。
// package.json を目印に上へ辿ってリポジトリルートを決める（column-view-init-source-scan.test.ts と同じ）
function find_repo_root(): string {
    let dir = process.cwd()
    for (let i = 0; i < 10; i++) {
        if (existsSync(join(dir, 'package.json')) && existsSync(join(dir, 'src', 'client'))) {
            return dir
        }
        const parent = dirname(dir)
        if (parent === dir) {
            break
        }
        dir = parent
    }
    throw new Error(`リポジトリルートが見つからない: cwd=${process.cwd()}`)
}

describe('dashboard-view.vue の地図の配線', () => {
    // テンプレートの配線を1行消しても型検査は通り、ピッカーで日を選んでも表示日が動かなくなる。
    // 地図は selected_date に従うので、requested_change_map_date は go_date へ配線されていなければならない
    it('GPSLogMap の requested_change_map_date は go_date へ配線されている', () => {
        const source = readFileSync(join(find_repo_root(), 'src/client/pages/views/dashboard-view.vue'), 'utf8')
        const start = source.indexOf('<GPSLogMap')
        expect(start, 'GPSLogMap が見つからない').toBeGreaterThan(-1)
        const end = source.indexOf('/>', start)
        expect(end, 'GPSLogMap の閉じが見つからない').toBeGreaterThan(start)
        const tag = source.slice(start, end)
        expect(
            /@requested_change_map_date="[^"]*\bgo_date\(/.test(tag),
            '地図の日付ピッカーが表示日（go_date）へ配線されていない',
        ).toBe(true)
    })
})
