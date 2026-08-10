/**
 * 設定画面の「適用を押すまで確定しない」不変条件の検証。
 *
 * 子ダイアログの適用ハンドラは clone にだけ書く。props.application_config を
 * 直接書き換えると、設定画面でキャンセルしても子ダイアログでの編集が残ってしまう
 * （Dnote / Ryuu / Dashboard / PlaingTimeIs の4つだけが props にも書いていた）。
 * ロケールとダークテーマは選ばせるために即時プレビューしているので、
 * 閉じるときに開いた時点の状態へ戻す必要がある。
 */
import { describe, expect, test, vi, beforeEach } from 'vitest'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させないと class extends が undefined になる
import '@/classes/api/gkill-api'

const theme_stub = { global: { name: { value: 'gkill_theme' } } }
const set_locale_mock = vi.fn()

vi.mock('vuetify', () => ({
    useTheme: () => theme_stub,
}))

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
    set_locale: (...args: Array<unknown>) => set_locale_mock(...args),
}))

// router は全ページを引き込むので、ログアウトで使う replace だけ差し替える
vi.mock('@/router', () => ({
    default: { replace: vi.fn() },
}))

vi.mock('@/classes/delete-gkill-cache', () => ({
    default: vi.fn().mockResolvedValue(undefined),
    delete_gkill_config_cache: vi.fn().mockResolvedValue(undefined),
}))

import { nextTick, reactive } from 'vue'
import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { useApplicationConfigView } from '@/classes/use-application-config-view'
import type { ApplicationConfigViewProps } from '@/pages/views/application-config-view-props'
import type { ApplicationConfigViewEmits } from '@/pages/views/application-config-view-emits'

const noop_emits = (() => { }) as unknown as ApplicationConfigViewEmits

function createProps(customize?: (config: ApplicationConfig) => void) {
    const application_config = new ApplicationConfig()
    customize?.(application_config)
    const gkill_api = {
        get_mi_board_list: vi.fn().mockResolvedValue({ boards: [], messages: [], errors: [] }),
        set_locale_name_to_cookie: vi.fn(),
        set_default_page_to_cookie: vi.fn(),
        update_application_config: vi.fn().mockResolvedValue({ messages: [], errors: [] }),
    }
    // props.application_config の差し替えを watch に拾わせるため reactive にする
    return reactive({
        application_config: application_config,
        gkill_api: gkill_api,
        app_content_height: 800,
        app_content_width: 1200,
    }) as unknown as ApplicationConfigViewProps & {
        application_config: ApplicationConfig,
        gkill_api: {
            set_locale_name_to_cookie: ReturnType<typeof vi.fn>,
            update_application_config: ReturnType<typeof vi.fn>,
        },
    }
}

beforeEach(() => {
    theme_stub.global.name.value = 'gkill_theme'
    set_locale_mock.mockClear()
})

describe('子ダイアログの適用ハンドラ', () => {
    test.each([
        {
            name: 'Dnote',
            apply: 'onRequestedApplyDnote',
            field: 'dnote_json_data',
        },
        {
            name: 'Ryuu',
            apply: 'onRequestedApplyRyuuStruct',
            field: 'ryuu_json_data',
        },
        {
            name: 'Dashboard',
            apply: 'onRequestedApplyDashboardStruct',
            field: 'dashboard_json_data',
        },
        {
            name: 'PlaingTimeIs',
            apply: 'onRequestedApplyPlaingTimeIs',
            field: 'plaing_timeis_json_data',
        },
        {
            name: 'SavedFindQuery',
            apply: 'onRequestedApplySavedFindQueryStruct',
            field: 'saved_find_query_json_data',
        },
    ])('$name の適用は clone にだけ書き、props を書き換えない', ({ apply, field }) => {
        const props = createProps()
        const view = useApplicationConfigView({ props, emits: noop_emits }) as unknown as
            Record<string, (data: Record<string, unknown>) => void> & {
                cloned_application_config: { value: Record<string, unknown> },
            }

        const props_fields = props.application_config as unknown as Record<string, unknown>
        const before = props_fields[field]

        const applied = { edited: true }
        view[apply](applied)

        // ref 越しなので同一性ではなく内容で見る
        expect(view.cloned_application_config.value[field], 'cloneに反映されていない').toStrictEqual(applied)
        expect(
            props_fields[field],
            'props を直接書き換えている（設定画面のキャンセルが効かなくなる）',
        ).toBe(before)
    })

    test('子ダイアログの適用ではサーバへ送らない', () => {
        const props = createProps()
        const view = useApplicationConfigView({ props, emits: noop_emits })

        view.onRequestedApplyTagStruct({ edited: true } as never)
        view.onRequestedApplyDnote({ edited: true })
        view.onRequestedApplyRyuuStruct({ edited: true })
        view.onRequestedApplyDashboardStruct({ edited: true })
        view.onRequestedApplyPlaingTimeIs({ edited: true })
        view.onRequestedApplySavedFindQueryStruct({ edited: true })

        expect(
            props.gkill_api.update_application_config,
            '子ダイアログの適用でサーバへ送っている（この画面の「適用」までは組み立てだけ）',
        ).not.toHaveBeenCalled()
    })

    test('この画面の適用で、子ダイアログの組み立て結果をまとめて送る', async () => {
        const props = createProps()
        const view = useApplicationConfigView({ props, emits: noop_emits })

        const dnote = { dnote: true }
        const dashboard = { dashboard: true }
        view.onRequestedApplyDnote(dnote)
        view.onRequestedApplyDashboardStruct(dashboard)

        // 送信後に sleep(1500) → location.reload() が走るので、タイマーを進めずに送信だけ見る
        vi.useFakeTimers()
        void view.update_application_config()
        await vi.advanceTimersByTimeAsync(0)
        vi.useRealTimers()

        expect(props.gkill_api.update_application_config).toHaveBeenCalledTimes(1)
        const sent = props.gkill_api.update_application_config.mock.calls[0][0]
        expect(sent.application_config.dnote_json_data).toStrictEqual(dnote)
        expect(sent.application_config.dashboard_json_data).toStrictEqual(dashboard)
    })
})

describe('未適用の編集の保持', () => {
    test('props が差し替わっても、子ダイアログで適用した内容は消えない', async () => {
        const props = createProps()
        const view = useApplicationConfigView({ props, emits: noop_emits })

        const applied = { edited: true }
        view.onRequestedApplyDnote(applied)

        // 板ツリー/タグツリーの追随などで props の identity が変わる
        props.application_config = new ApplicationConfig()
        await nextTick()

        expect(
            view.cloned_application_config.value.dnote_json_data,
            'props の差し替えで未適用の編集が消えている',
        ).toStrictEqual(applied)
    })

    test('未適用の編集が無ければ props の差し替えに追随する', async () => {
        const props = createProps()
        const view = useApplicationConfigView({ props, emits: noop_emits })

        const loaded = new ApplicationConfig()
        loaded.google_map_api_key = 'loaded-later'
        props.application_config = loaded
        await nextTick()

        expect(view.google_map_api_key.value, '後から届いた設定に追随していない').toBe('loaded-later')
    })
})

describe('cancel_pending_changes', () => {
    test('ダークテーマを開いた時点へ戻す', async () => {
        const props = createProps()
        const { use_dark_theme, cancel_pending_changes } = useApplicationConfigView({ props, emits: noop_emits })

        use_dark_theme.value = true
        await nextTick()
        expect(theme_stub.global.name.value).toBe('gkill_dark_theme')

        cancel_pending_changes()
        await nextTick()
        expect(theme_stub.global.name.value, 'キャンセルでテーマが戻らない').toBe('gkill_theme')
    })

    test('ロケールを開いた時点へ戻す', async () => {
        const props = createProps()
        const { locale_name, cancel_pending_changes } = useApplicationConfigView({ props, emits: noop_emits })

        locale_name.value = 'en'
        await nextTick()
        expect(set_locale_mock).toHaveBeenLastCalledWith('en')

        cancel_pending_changes()
        await nextTick()
        expect(set_locale_mock, 'キャンセルでロケールが戻らない').toHaveBeenLastCalledWith('ja')
        expect(props.gkill_api.set_locale_name_to_cookie).toHaveBeenLastCalledWith('ja')
    })

    test('変えていなければ何もしない', async () => {
        const props = createProps()
        const { cancel_pending_changes } = useApplicationConfigView({ props, emits: noop_emits })

        cancel_pending_changes()
        await nextTick()
        expect(set_locale_mock).not.toHaveBeenCalled()
        expect(theme_stub.global.name.value).toBe('gkill_theme')
    })
})

describe('reload_cloned_application_config', () => {
    test('保存済みの日数を既定値で潰さない', async () => {
        // 期間を使う設定で保存されている状態
        const props = createProps((config) => {
            config.rykv_default_period = 60
            config.mi_default_period = 90
        })
        const view = useApplicationConfigView({ props, emits: noop_emits })

        await view.reload_cloned_application_config()

        expect(view.is_checked_use_rykv_period.value, 'チェックが復元されていない').toBe(true)
        expect(view.rykv_default_period.value, 'チェックのwatcherが保存済みの日数を31で潰している').toBe(60)
        expect(view.is_checked_use_mi_period.value).toBe(true)
        expect(view.mi_default_period.value).toBe(90)
    })

    test('期間を使わない設定では -1 のまま', async () => {
        const props = createProps((config) => {
            config.rykv_default_period = -1
            config.mi_default_period = -1
        })
        const view = useApplicationConfigView({ props, emits: noop_emits })

        await view.reload_cloned_application_config()

        expect(view.is_checked_use_rykv_period.value).toBe(false)
        expect(view.rykv_default_period.value).toBe(-1)
    })

    test('開き直したあとのキャンセルは開き直した時点へ戻す', async () => {
        const props = createProps()
        const view = useApplicationConfigView({ props, emits: noop_emits })

        // 1回目: ダークテーマにして開き直す（開き直しは適用ではないので、テーマは効いたまま）
        view.use_dark_theme.value = true
        await nextTick()
        await view.reload_cloned_application_config()
        expect(theme_stub.global.name.value).toBe('gkill_dark_theme')

        // 2回目: 明るいテーマへ戻してキャンセル → 開き直した時点（ダーク）に戻る
        view.use_dark_theme.value = false
        await nextTick()
        view.cancel_pending_changes()
        await nextTick()
        expect(theme_stub.global.name.value).toBe('gkill_dark_theme')
    })
})
