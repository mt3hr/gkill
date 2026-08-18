/**
 * useDashboardPage は ApplicationConfig の取得・テーマ・メッセージ表示・
 * 板ツリー/タグツリーの追随・ログアウトだけを持つ薄いページ。
 * 一覧の更新や引き直しは useDashboardView 側（dashboard-view-reload.test.ts）。
 */
import { describe, expect, test, vi, beforeEach, afterEach } from 'vitest'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'

vi.mock('vuetify', () => ({
    useTheme: () => ({ global: { name: { value: 'gkill_theme' } } }),
}))
vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))
// router は全ページを引き込むので、この画面が使う replace だけ差し替える
vi.mock('@/router', () => ({
    default: { replace: vi.fn(), push: vi.fn() },
}))
vi.mock('vue-router', () => ({
    useRoute: () => ({ path: '/dashboard', query: {}, params: {} }),
    useRouter: () => ({ replace: vi.fn(), push: vi.fn() }),
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

import { createApp, defineComponent, h } from 'vue'
import { useDashboardPage } from '@/classes/use-dashboard-page'
import { GkillAPI } from '@/classes/api/gkill-api'
import { ApplicationConfig } from '@/classes/datas/config/application-config'

function make_fake_api() {
    return {
        get_session_id: vi.fn(() => 'test-session'),
        generate_uuid: vi.fn(() => 'generated-uuid'),
        get_application_config: vi.fn().mockResolvedValue({
            application_config: new ApplicationConfig(),
            messages: null,
            errors: null,
        }),
        set_use_dark_theme: vi.fn(),
        set_saved_application_config: vi.fn(),
        get_all_tag_names: vi.fn().mockResolvedValue({ messages: null, errors: null }),
        get_mi_board_list: vi.fn().mockResolvedValue({ boards: [], messages: null, errors: null }),
    }
}

let mounted_apps = new Array<ReturnType<typeof createApp>>()

function mount_page() {
    let page: ReturnType<typeof useDashboardPage> | null = null
    const Host = defineComponent({
        setup() {
            page = useDashboardPage()
            return () => h('div')
        },
    })
    const app = createApp(Host)
    app.mount(document.createElement('div'))
    mounted_apps.push(app)
    return { app: app, page: page! }
}

beforeEach(() => {
    vi.useFakeTimers()
    vi.spyOn(GkillAPI, 'get_instance').mockReturnValue(make_fake_api() as unknown as GkillAPI)
})

afterEach(() => {
    for (const app of mounted_apps) {
        app.unmount()
    }
    mounted_apps = []
    vi.useRealTimers()
    vi.restoreAllMocks()
})

/**
 * 設定が取れないと application_config.is_loaded が立たず、画面の初期化が
 * 一度も走らない。黙って戻ると読み込み中オーバーレイのまま永久に固まるので、
 * 失敗を画面へ伝えて再試行できるようにする。
 */
describe('ApplicationConfig 取得の失敗', () => {
    test('errors が返ったら application_config_load_failed が立ち、設定は差し替わらない', async () => {
        const api = make_fake_api()
        api.get_application_config = vi.fn().mockResolvedValue({
            application_config: new ApplicationConfig(),
            messages: null,
            errors: [{ error_code: 'ERR000001', error_message: '設定が取れない' }],
        })
        vi.spyOn(GkillAPI, 'get_instance').mockReturnValue(api as unknown as GkillAPI)

        const { page } = mount_page()
        await vi.advanceTimersByTimeAsync(0)

        expect(page.application_config_load_failed.value, '失敗が画面へ伝わっていない').toBe(true)
        expect(page.application_config.value.is_loaded, '失敗したのに設定が差し替わっている').toBeFalsy()
    })

    test('通信例外でも application_config_load_failed が立つ', async () => {
        // catch が無いと呼び出し元が await していないぶん unhandled rejection になり、
        // やはり画面が固まったままになる
        const api = make_fake_api()
        api.get_application_config = vi.fn().mockRejectedValue(new Error('network down'))
        vi.spyOn(GkillAPI, 'get_instance').mockReturnValue(api as unknown as GkillAPI)
        const console_error = vi.spyOn(console, 'error').mockImplementation(() => { })

        const { page } = mount_page()
        await vi.advanceTimersByTimeAsync(0)

        expect(page.application_config_load_failed.value, '例外が画面へ伝わっていない').toBe(true)
        console_error.mockRestore()
    })

    // 再試行ボタンはコンポーネントインスタンスの外から load_application_config を呼ぶ。
    // useRoute() を関数の中で呼んでいると、そこで inject が undefined を返して落ちる
    // （＝「永久スピナーにしない」導線がそこで壊れる）ので、setup の中で1回だけ呼ぶこと
    test('再試行が成功したら失敗フラグは倒れる', async () => {
        const api = make_fake_api()
        const loaded_config = new ApplicationConfig()
        loaded_config.is_loaded = true
        api.get_application_config = vi.fn()
            .mockResolvedValueOnce({
                application_config: new ApplicationConfig(),
                messages: null,
                errors: [{ error_code: 'ERR000001', error_message: '設定が取れない' }],
            })
            .mockResolvedValue({ application_config: loaded_config, messages: null, errors: null })
        vi.spyOn(GkillAPI, 'get_instance').mockReturnValue(api as unknown as GkillAPI)

        const { page } = mount_page()
        await vi.advanceTimersByTimeAsync(0)
        expect(page.application_config_load_failed.value).toBe(true)

        await page.load_application_config()
        await vi.advanceTimersByTimeAsync(0)

        expect(page.application_config_load_failed.value, '再試行しても失敗表示のままになる').toBe(false)
        expect(page.application_config.value.is_loaded).toBe(true)
    })
})

describe('DashboardView からのイベント中継', () => {
    // ビューは一覧の更新を自分で済ませたうえで registered_kyou / updated_kyou を上げてくる。
    // ページの仕事は板ツリー/タグツリーの追随とメッセージ表示だけ
    test('received_errors はスナックバーのメッセージ列へ積む', async () => {
        const { page } = mount_page()

        page.dashboardViewHandlers.received_errors([
            { error_code: 'ERR000001', error_message: 'テストエラー', show_keep: true },
        ] as unknown as Parameters<typeof page.dashboardViewHandlers.received_errors>[0])
        await vi.advanceTimersByTimeAsync(0)

        expect(page.messages.value).toHaveLength(1)
        expect(page.messages.value[0].is_error).toBe(true)
    })
})
