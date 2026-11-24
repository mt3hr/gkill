/**
 * プラグイン一覧ダイアログ（use-manage-plugin-list-dialog）。ADR-0714。
 *
 * 管理者がアカウント管理画面の行から開く画面で、**どのアカウントのプラグインを見るかは
 * 開いた行で決まる**。プラグインは利用者ごとの実体なので、`target_user_id` を載せ忘れると、
 * エラーも警告も出ないまま管理者自身のプラグインが他人の行に並ぶ。
 * 一覧と、そこから開く設定画面の両方で固定する。
 */
import { describe, expect, test, vi } from 'vitest'

vi.mock('@/i18n', () => ({
    i18n: { global: { t: (key: string) => key, locale: 'ja' } },
}))
vi.mock('@/classes/use-dialog-history-stack', () => ({
    useDialogHistoryStack: vi.fn(),
    close_dialog_via_history: vi.fn(),
}))
vi.mock('@/classes/use-floating-dialog', () => ({
    useFloatingDialog: vi.fn(() => ({})),
}))

// req_res は GkillAPIRequest を継承する。循環importがあるため gkill-api を先に評価させる
import '@/classes/api/gkill-api'

import { useManagePluginListDialog } from '@/classes/use-manage-plugin-list-dialog'
import { Account } from '@/classes/datas/config/account'
import type { ManagePluginListDialogProps } from '@/pages/dialogs/manage-plugin-list-dialog-props'
import type { ManagePluginListDialogEmits } from '@/pages/dialogs/manage-plugin-list-dialog-emits'
import type { GetPluginListRequest } from '@/classes/api/req_res/get-plugin-list-request'
import type { PluginInfo } from '@/classes/api/req_res/get-plugin-list-response'

function ok<T extends object>(extra: T) {
    return { errors: [], messages: [], ...extra }
}

function plugin_of(name: string, rep_name: string): PluginInfo {
    return { name: name, version: '1.0.0', description: 'd', rep_name: rep_name, is_alive: true } as unknown as PluginInfo
}

function account_of(user_id: string): Account {
    const account = new Account()
    account.user_id = user_id
    return account
}

function create_dialog(api_overrides: Record<string, unknown> = {}) {
    const api = {
        get_plugin_list: vi.fn(async (_req: GetPluginListRequest) => ok({ plugins: [plugin_of('gkill_plugin_example', 'Example')] })),
        ...api_overrides,
    }
    const props = { application_config: {}, gkill_api: api } as unknown as ManagePluginListDialogProps
    const emitted: Array<{ event: string, payload: unknown }> = []
    const emits = ((event: string, payload: unknown) => { emitted.push({ event, payload }) }) as unknown as ManagePluginListDialogEmits
    const dialog = useManagePluginListDialog({ props, emits })
    const plugin_config_show = vi.fn()
    dialog.plugin_config_dialog.value = { show: plugin_config_show } as unknown as typeof dialog.plugin_config_dialog.value
    return { dialog, api, emitted, plugin_config_show }
}

describe('useManagePluginListDialog', () => {
    test('開いた行のアカウントを対象に1回だけ一覧を読み込む', async () => {
        const { dialog, api } = create_dialog()
        expect(api.get_plugin_list).not.toHaveBeenCalled()

        await dialog.show(account_of('testuser'))

        expect(api.get_plugin_list).toHaveBeenCalledTimes(1)
        expect(api.get_plugin_list.mock.calls[0][0].target_user_id).toBe('testuser')
        expect(dialog.plugins.value.length).toBe(1)
        expect(dialog.is_show_dialog.value).toBe(true)
        expect(dialog.is_loading.value).toBe(false)
    })

    test('エラーはフィードへ流し、一覧を埋めない', async () => {
        const { dialog, emitted } = create_dialog({
            get_plugin_list: vi.fn(async () => ({
                messages: [], plugins: null,
                errors: [{ error_code: 'ERR000014', error_message: '管理者権限を所有していません' }],
            })),
        })

        await dialog.show(account_of('testuser'))

        expect(emitted.filter((e) => e.event === 'received_errors').length).toBe(1)
        expect(dialog.plugins.value.length).toBe(0)
        // 読み込み表示を立てたまま終わらない（finally で必ず倒す）
        expect(dialog.is_loading.value).toBe(false)
    })

    // 閉じたときに残すと、次に別のアカウントの行から開いた瞬間、
    // 取り直しが終わるまで前のアカウントのプラグインが並ぶ
    test('閉じたら一覧と対象アカウントを空に戻す', async () => {
        const { dialog } = create_dialog()
        await dialog.show(account_of('testuser'))

        await dialog.hide()

        expect(dialog.plugins.value.length).toBe(0)
        expect(dialog.cloned_account.value.user_id).toBe('')
    })

    test('設定画面へ対象利用者と rep 名の両方を渡す', async () => {
        const { dialog, plugin_config_show } = create_dialog()
        await dialog.show(account_of('testuser'))

        dialog.show_plugin_config_dialog(plugin_of('gkill_plugin_example', 'Example'))

        expect(plugin_config_show).toHaveBeenCalledWith('testuser', 'Example')
    })
})
