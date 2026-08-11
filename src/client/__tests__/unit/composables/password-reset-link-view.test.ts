/**
 * パスワードリセットリンク表示ダイアログの検証。
 *
 * リセットトークンは発行から72時間で期限切れになるが、管理画面は
 * トークンがある間このダイアログしか出さない。ここに再発行がないと、
 * 期限切れから復帰する手段がサーバと同じマシンでの `reset_password`
 * コマンドだけになる（実際にそうなっていた）。
 * また、URLに user_id が載っていないと利用者がユーザ名を手入力することになり、
 * 打ち間違えると「トークンは正しいのに失敗する」状態になる。
 */
import { describe, expect, test, vi } from 'vitest'

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させる
import '@/classes/api/gkill-api'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))

import { createApp, defineComponent, h, nextTick, reactive } from 'vue'
import { Account } from '@/classes/datas/config/account'
import { useShowPasswordResetLinkView } from '@/classes/use-show-password-reset-link-view'
import { useManageAccountView } from '@/classes/use-manage-account-view'
import type { ShowPasswordResetLinkViewProps } from '@/pages/views/show-password-reset-link-view-props'
import type { ShowPasswordResetLinkViewEmits } from '@/pages/views/show-password-reset-link-view-emits'
import type { ManageAccountViewProps } from '@/pages/views/manage-account-view-props'
import type { ManageAccountViewEmits } from '@/pages/views/manage-account-view-emits'

interface EmittedEvent {
    event: string
    args: Array<unknown>
}

// サーバから来るのはメソッドを持たない生JSONで、Account に詰め直されるわけではない。
// ここでも Object.assign で組み立てて本番の形に合わせる
function make_account(options: { user_id: string, token: string | null, expiration: string | null }): Account {
    return Object.assign(new Account(), {
        user_id: options.user_id,
        is_admin: false,
        is_enable: true,
        password_reset_token: options.token,
        password_reset_token_expiration: options.expiration,
    })
}

function create_link_view(options: {
    account: Account,
    reset_password?: ReturnType<typeof vi.fn>,
}) {
    const emitted = new Array<EmittedEvent>()
    const emits = ((event: string, ...args: Array<unknown>) => {
        emitted.push({ event: event, args: args })
    }) as unknown as ShowPasswordResetLinkViewEmits
    const props = reactive({
        account: options.account,
        server_configs: [{
            enable_this_device: true,
            enable_tls: true,
            address: ':9999',
            lan_hostname: '',
            global_hostname: '',
        }],
        application_config: { private_ip: '192.168.0.1' },
        gkill_api: {
            reset_password: options.reset_password
                ?? vi.fn().mockResolvedValue({ messages: null, errors: null, password_reset_path_without_host: 'new-token' }),
        },
    }) as unknown as ShowPasswordResetLinkViewProps

    // onUnmounted を使うのでコンポーネントの中で呼ぶ
    let view: ReturnType<typeof useShowPasswordResetLinkView> | null = null
    const Host = defineComponent({
        setup() {
            view = useShowPasswordResetLinkView({ props: props, emits: emits })
            return () => h('div')
        },
    })
    const container = document.createElement('div')
    const app = createApp(Host)
    app.mount(container)
    return { view: view!, props: props, emitted: emitted, app: app }
}

function events_of(emitted: Array<EmittedEvent>, event: string): Array<EmittedEvent> {
    return emitted.filter((entry) => entry.event === event)
}

describe('リセットURL', () => {
    test('user_id を載せるので設定画面のユーザ名欄が手入力にならない', () => {
        const { view, app } = create_link_view({
            account: make_account({ user_id: 'test', token: 'token-1', expiration: '2026-08-15T04:55:16+09:00' }),
        })

        expect(view.lan_password_reset_url.value)
            .toBe('https://192.168.0.1:9999/set_new_password?user_id=test&reset_token=token-1')
        expect(view.local_password_reset_url.value)
            .toBe('https://localhost:9999/set_new_password?user_id=test&reset_token=token-1')
        app.unmount()
    })
})

describe('有効期限', () => {
    test('期限を表示する', () => {
        const { view, app } = create_link_view({
            account: make_account({ user_id: 'test', token: 'token-1', expiration: '2026-08-15T04:55:16+09:00' }),
        })

        expect(view.password_reset_token_expiration_label.value).not.toBe('')
        app.unmount()
    })

    test('期限が過ぎていたら期限切れとして扱う', () => {
        const past = new Date(Date.now() - 60 * 60 * 1000).toISOString()
        const { view, app } = create_link_view({
            account: make_account({ user_id: 'test', token: 'token-1', expiration: past }),
        })

        expect(view.is_password_reset_link_expired.value).toBe(true)
        app.unmount()
    })

    test('期限内なら期限切れにしない', () => {
        const future = new Date(Date.now() + 60 * 60 * 1000).toISOString()
        const { view, app } = create_link_view({
            account: make_account({ user_id: 'test', token: 'token-1', expiration: future }),
        })

        expect(view.is_password_reset_link_expired.value).toBe(false)
        app.unmount()
    })

    test('期限が無ければ表示も期限切れ判定もしない', () => {
        const { view, app } = create_link_view({
            account: make_account({ user_id: 'test', token: 'token-1', expiration: null }),
        })

        expect(view.password_reset_token_expiration_label.value).toBe('')
        expect(view.is_password_reset_link_expired.value).toBe(false)
        app.unmount()
    })
})

describe('リンクの再発行', () => {
    test('reset_password を叩き、親に引き直しと再表示を頼む', async () => {
        const reset_password = vi.fn().mockResolvedValue({ messages: null, errors: null, password_reset_path_without_host: 'new-token' })
        const account = make_account({ user_id: 'test', token: 'old-token', expiration: '2026-08-06T06:59:20+09:00' })
        const { view, emitted, app } = create_link_view({ account: account, reset_password: reset_password })

        await view.reissue_password_reset_link()
        await nextTick()

        expect(reset_password).toHaveBeenCalledTimes(1)
        expect(reset_password.mock.calls[0][0].target_user_id).toBe('test')
        // 新しいトークンは親がサーバから引き直して渡しなおす。
        // レスポンスのトークンを自分で持つと有効期限だけ古いままになる
        expect(events_of(emitted, 'requested_reload_server_config')).toHaveLength(1)
        expect(events_of(emitted, 'requested_show_show_password_reset_dialog')).toHaveLength(1)
        app.unmount()
    })

    test('失敗したらエラーを上げ、再表示は頼まない', async () => {
        const reset_password = vi.fn().mockResolvedValue({
            messages: null,
            errors: [{ error_code: 'ERR000014', error_message: 'no auth' }],
            password_reset_path_without_host: '',
        })
        const { view, emitted, app } = create_link_view({
            account: make_account({ user_id: 'test', token: 'old-token', expiration: null }),
            reset_password: reset_password,
        })

        await view.reissue_password_reset_link()
        await nextTick()

        expect(events_of(emitted, 'received_errors')).toHaveLength(1)
        expect(events_of(emitted, 'requested_show_show_password_reset_dialog')).toHaveLength(0)
        app.unmount()
    })

    test('連打しても1回しか発行しない', async () => {
        let resolve_reset: ((value: unknown) => void) | null = null
        const reset_password = vi.fn().mockImplementation(() => new Promise((resolve) => { resolve_reset = resolve }))
        const { view, app } = create_link_view({
            account: make_account({ user_id: 'test', token: 'old-token', expiration: null }),
            reset_password: reset_password,
        })

        const first = view.reissue_password_reset_link()
        await view.reissue_password_reset_link()
        resolve_reset!({ messages: null, errors: null, password_reset_path_without_host: 'new-token' })
        await first

        expect(reset_password).toHaveBeenCalledTimes(1)
        app.unmount()
    })
})

describe('自分自身のアカウント', () => {
    function create_manage_view(login_user_id: string) {
        const emits = (() => { /* 使わない */ }) as unknown as ManageAccountViewEmits
        const props = reactive({
            server_configs: [{
                enable_this_device: true,
                accounts: [
                    make_account({ user_id: 'admin', token: null, expiration: null }),
                    make_account({ user_id: 'test', token: 'token-1', expiration: null }),
                ],
            }],
            application_config: { user_id: login_user_id },
            gkill_api: {},
        }) as unknown as ManageAccountViewProps
        return useManageAccountView({ props: props, emits: emits })
    }

    test('ログイン中のアカウントは有効・無効を切り替えさせない', () => {
        const view = create_manage_view('admin')

        expect(view.is_own_account(make_account({ user_id: 'admin', token: null, expiration: null }))).toBe(true)
        expect(view.is_own_account(make_account({ user_id: 'test', token: null, expiration: null }))).toBe(false)
    })
})
