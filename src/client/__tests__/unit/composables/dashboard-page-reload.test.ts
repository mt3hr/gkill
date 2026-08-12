/**
 * useDashboardPage の再読込まわりの検証（ページ全機能は狙わない）。
 *
 * 1. registered_kyou のデバウンス。KFTLで複数行を一度に投げると registered_kyou が
 *    連続発火するので、300ms まとめて1回だけ取り直す。まとめないと行数ぶん全体検索が走る。
 * 2. アンマウント後にタイマーが残らないこと。残ると画面を離れたあとに検索が飛ぶ。
 * 3. reload_kyou は Mi リスト / チェック済み / 開いているダイアログの3箇所を、
 *    同じ requested_at で引き直す。同じ値を渡さないと3系統が別々に往復する。
 * 4. dashboardKyouHandlers の結線。以前は RykvDialogHost に closed と
 *    received_* しか配線しておらず、何を編集しても画面が更新されなかった。
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
vi.mock('@/classes/use-scoped-enter-for-kftl', () => ({ useScopedEnterForKFTL: vi.fn() }))
vi.mock('@/classes/use-scoped-ctrl-v-for-clipboard', () => ({ useScopedCtrlVForClipboard: vi.fn() }))
// 引き直しの手順そのものは kyou-reload.test.ts が見る。ここでは呼ばれ方だけ見る
vi.mock('@/classes/kyou-reload', () => ({
    new_reload_batch: vi.fn(),
    refresh_kyou: vi.fn(),
    refresh_kyou_in_list: vi.fn(),
    build_mi_reload_query: vi.fn((query: unknown) => query),
}))

import { createApp, defineComponent, h } from 'vue'
import { useDashboardPage } from '@/classes/use-dashboard-page'
import { GkillAPI } from '@/classes/api/gkill-api'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { new_reload_batch, refresh_kyou, refresh_kyou_in_list } from '@/classes/kyou-reload'
import type { Kyou } from '@/classes/datas/kyou'
import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { OpenedRykvDialog } from '@/pages/views/rykv-dialog-kind'

const new_reload_batch_mock = vi.mocked(new_reload_batch)
const refresh_kyou_mock = vi.mocked(refresh_kyou)
const refresh_kyou_in_list_mock = vi.mocked(refresh_kyou_in_list)

interface RefreshInListOptions {
    requested_at?: number
    query?: unknown
    replace?: (next_list: Array<Kyou>) => void
}

/** Kyou の実クラスは通信を伴うので、この画面が触るフィールドだけの構造フェイクを使う */
function make_kyou(id: string, data_type = 'kmemo'): Kyou {
    return {
        id: id,
        data_type: data_type,
        related_time: new Date('2026-03-15T09:00:00+09:00'),
        attached_tags: [],
    } as unknown as Kyou
}

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
        get_kyous: vi.fn().mockResolvedValue({ kyous: [], messages: null, errors: null }),
        get_kyou: vi.fn().mockResolvedValue({ kyou_histories: [], messages: null, errors: null }),
        get_all_tag_names: vi.fn().mockResolvedValue({ messages: null, errors: null }),
        get_mi_board_list: vi.fn().mockResolvedValue({ boards: [], messages: null, errors: null }),
    }
}

let mounted_apps = new Array<ReturnType<typeof createApp>>()

function mount_page(options?: { reload_all?: () => Promise<void> }) {
    let page: ReturnType<typeof useDashboardPage> | null = null
    const Host = defineComponent({
        setup() {
            page = useDashboardPage(options)
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
    // 引き直しのバッチ時刻は連番にして、3箇所へ同じ値が渡っていることを見分けられるようにする
    let batch_counter = 0
    new_reload_batch_mock.mockReset()
    new_reload_batch_mock.mockImplementation(() => ++batch_counter)
    refresh_kyou_mock.mockReset()
    refresh_kyou_mock.mockResolvedValue(null)
    refresh_kyou_in_list_mock.mockReset()
    refresh_kyou_in_list_mock.mockResolvedValue(undefined)
})

afterEach(() => {
    for (const app of mounted_apps) {
        app.unmount()
    }
    mounted_apps = []
    vi.useRealTimers()
    vi.restoreAllMocks()
})

describe('registered_kyou のデバウンス', () => {
    test('連続して登録されても再読込は1回にまとまる', async () => {
        const reload_all = vi.fn().mockResolvedValue(undefined)
        const { page } = mount_page({ reload_all: reload_all })

        page.dashboardKyouHandlers.registered_kyou(make_kyou('kyou-1'))
        page.dashboardKyouHandlers.registered_kyou(make_kyou('kyou-2'))

        await vi.advanceTimersByTimeAsync(299)
        expect(reload_all, '300ms 経つ前に取り直している').not.toHaveBeenCalled()

        await vi.advanceTimersByTimeAsync(1)
        expect(reload_all, '登録のたびに全体検索が走っている（まとめられていない）').toHaveBeenCalledTimes(1)
    })

    test('間隔を空けた登録はそれぞれ取り直す', async () => {
        const reload_all = vi.fn().mockResolvedValue(undefined)
        const { page } = mount_page({ reload_all: reload_all })

        page.dashboardKyouHandlers.registered_kyou(make_kyou('kyou-1'))
        await vi.advanceTimersByTimeAsync(300)
        page.dashboardKyouHandlers.registered_kyou(make_kyou('kyou-2'))
        await vi.advanceTimersByTimeAsync(300)

        expect(reload_all).toHaveBeenCalledTimes(2)
    })

    test('アンマウント後はタイマーが残らない', async () => {
        const reload_all = vi.fn().mockResolvedValue(undefined)
        const { app, page } = mount_page({ reload_all: reload_all })

        page.dashboardKyouHandlers.registered_kyou(make_kyou('kyou-1'))
        app.unmount()

        await vi.advanceTimersByTimeAsync(1000)
        expect(reload_all, '画面を離れたあとに検索が飛んでいる').not.toHaveBeenCalled()
    })
})

describe('reload_kyou', () => {
    test('Mi リスト / チェック済み / 開いているダイアログを同じ requested_at で引き直す', async () => {
        const { page } = mount_page()
        const kyou = make_kyou('kyou-1', 'mi_create')
        page.mi_kyous.value = [make_kyou('kyou-1', 'mi_create')]
        page.checked_kyous.value = [make_kyou('kyou-1', 'mi_create')]
        page.opened_dialogs.value = [{
            id: 'dialog-1',
            kind: 'kyou',
            kyou: make_kyou('kyou-1', 'mi_create'),
            payload: null,
            opened_at: 0,
        } as OpenedRykvDialog]
        const refreshed = make_kyou('kyou-1', 'mi_create')
        refresh_kyou_mock.mockResolvedValue(refreshed)

        await page.reload_kyou(kyou)

        expect(new_reload_batch_mock, '1回の更新でバッチ時刻を取り直している').toHaveBeenCalledTimes(1)
        const batch = new_reload_batch_mock.mock.results[0].value as number

        expect(refresh_kyou_in_list_mock, 'Mi リストとチェック済みの2箇所を引き直していない').toHaveBeenCalledTimes(2)
        for (let i = 0; i < 2; i++) {
            const options = refresh_kyou_in_list_mock.mock.calls[i][2] as RefreshInListOptions
            expect(options.requested_at, 'リストの引き直しが別バッチになっている（合流せず往復が増える）').toBe(batch)
        }

        expect(refresh_kyou_mock, '開いているダイアログを引き直していない').toHaveBeenCalledTimes(1)
        expect(
            refresh_kyou_mock.mock.calls[0][2],
            'ダイアログの引き直しが別バッチになっている',
        ).toBe(batch)
    })

    test('Mi リストの引き直しには Mi 用の検索条件を渡す（並び順が変わらないように）', async () => {
        const { page } = mount_page()
        page.mi_kyous.value = [make_kyou('kyou-1', 'mi_create')]

        await page.reload_kyou(make_kyou('kyou-1', 'mi_create'))

        const options = refresh_kyou_in_list_mock.mock.calls[0][2] as RefreshInListOptions
        const query = options.query as FindKyouQuery
        expect(query, 'Mi リストの引き直しに検索条件を渡していない').toBeTruthy()
        expect(query.for_mi).toBe(true)
    })

    test('Mi リストは replace で copy-on-write する', async () => {
        const { page } = mount_page()
        page.mi_kyous.value = [make_kyou('kyou-1', 'mi_create')]
        const next_list = [make_kyou('kyou-1', 'mi_create')]
        refresh_kyou_in_list_mock.mockImplementation(async (_list, _kyou, options) => {
            const replace = (options as RefreshInListOptions | undefined)?.replace
            if (replace) {
                replace(next_list)
            }
        })

        await page.reload_kyou(make_kyou('kyou-1', 'mi_create'))

        expect(page.mi_kyous.value, 'replace が配線されていない（列が再描画されない）').toHaveLength(1)
    })

    test('開いているダイアログの Kyou を引き直した結果で差し替える', async () => {
        const { page } = mount_page()
        page.opened_dialogs.value = [{
            id: 'dialog-1',
            kind: 'kyou',
            kyou: make_kyou('kyou-1'),
            payload: null,
            opened_at: 0,
        } as OpenedRykvDialog]
        const refreshed = make_kyou('kyou-1')
        refresh_kyou_mock.mockResolvedValue(refreshed)

        await page.reload_kyou(make_kyou('kyou-1'))

        expect(page.opened_dialogs.value[0].kyou.id).toBe('kyou-1')
        expect(page.opened_dialogs.value[0].id, 'ダイアログ自体を作り直してはいけない').toBe('dialog-1')
    })

    test('id が違うダイアログは引き直さない', async () => {
        const { page } = mount_page()
        page.opened_dialogs.value = [{
            id: 'dialog-1',
            kind: 'kyou',
            kyou: make_kyou('kyou-other'),
            payload: null,
            opened_at: 0,
        } as OpenedRykvDialog]

        await page.reload_kyou(make_kyou('kyou-1'))

        expect(refresh_kyou_mock, '無関係なダイアログまで引き直している').not.toHaveBeenCalled()
    })
})

describe('dashboardKyouHandlers の結線', () => {
    test('updated_kyou で該当 Kyou を引き直す', async () => {
        const { page } = mount_page()
        page.mi_kyous.value = [make_kyou('kyou-1')]

        page.dashboardKyouHandlers.updated_kyou(make_kyou('kyou-1'))
        await vi.advanceTimersByTimeAsync(0)

        expect(refresh_kyou_in_list_mock, '編集しても画面が更新されない').toHaveBeenCalled()
        expect((refresh_kyou_in_list_mock.mock.calls[0][1] as Kyou).id).toBe('kyou-1')
    })

    // タグ/テキスト/通知の変更は updated_kyou を出さない。唯一の信号がこれ
    test('requested_reload_kyou でも引き直す', async () => {
        const { page } = mount_page()
        page.mi_kyous.value = [make_kyou('kyou-1')]

        page.dashboardKyouHandlers.requested_reload_kyou(make_kyou('kyou-1'))
        await vi.advanceTimersByTimeAsync(0)

        expect(refresh_kyou_in_list_mock, 'タグを足しても表示が変わらない').toHaveBeenCalled()
    })

    test('deleted_kyou で3つのリストから消える', async () => {
        const { page } = mount_page()
        page.mi_kyous.value = [make_kyou('kyou-1'), make_kyou('kyou-2')]
        page.checked_kyous.value = [make_kyou('kyou-1')]
        page.dnote_kyous.value = [make_kyou('kyou-1')]

        page.dashboardKyouHandlers.deleted_kyou(make_kyou('kyou-1'))
        await vi.advanceTimersByTimeAsync(0)

        expect(page.mi_kyous.value.map((kyou) => kyou.id)).toEqual(['kyou-2'])
        expect(page.checked_kyous.value).toHaveLength(0)
        expect(page.dnote_kyous.value).toHaveLength(0)
    })

    test('requested_reload_list は待たずに取り直す（デバウンスしない）', async () => {
        const reload_all = vi.fn().mockResolvedValue(undefined)
        const { page } = mount_page({ reload_all: reload_all })

        page.dashboardKyouHandlers.requested_reload_list()
        await vi.advanceTimersByTimeAsync(0)

        expect(reload_all, '明示的な再検索まで300ms待たされている').toHaveBeenCalledTimes(1)
    })

    test('closed で開いているダイアログを閉じる', async () => {
        const { page } = mount_page()
        page.opened_dialogs.value = [{
            id: 'dialog-1',
            kind: 'kyou',
            kyou: make_kyou('kyou-1'),
            payload: null,
            opened_at: 0,
        } as OpenedRykvDialog]

        page.dashboardKyouHandlers.closed('dialog-1')

        expect(page.opened_dialogs.value).toHaveLength(0)
    })

    test('requested_update_check_kyous でチェック済みが増減する', async () => {
        const { page } = mount_page()
        const kyou = make_kyou('kyou-1')

        page.dashboardKyouHandlers.requested_update_check_kyous([kyou], true)
        expect(page.checked_kyous.value.map((checked) => checked.id)).toEqual(['kyou-1'])

        page.dashboardKyouHandlers.requested_update_check_kyous([kyou], false)
        expect(page.checked_kyous.value).toHaveLength(0)
    })

    test('received_errors はスナックバーのメッセージ列へ積む', async () => {
        const { page } = mount_page()

        page.dashboardKyouHandlers.received_errors([
            { error_code: 'ERR000001', error_message: 'テストエラー', show_keep: true },
        ] as unknown as Parameters<typeof page.dashboardKyouHandlers.received_errors>[0])
        await vi.advanceTimersByTimeAsync(0)

        expect(page.messages.value).toHaveLength(1)
        expect(page.messages.value[0].is_error).toBe(true)
    })
})
