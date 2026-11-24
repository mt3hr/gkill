'use strict'

import { type Ref, ref } from 'vue'
import type { ManagePluginListDialogEmits } from '@/pages/dialogs/manage-plugin-list-dialog-emits'
import type { ManagePluginListDialogProps } from '@/pages/dialogs/manage-plugin-list-dialog-props'
import type PluginConfigDialog from '@/pages/dialogs/plugin-config-dialog.vue'
import type HelpDialog from '@/pages/dialogs/help-dialog.vue'
import { Account } from '@/classes/datas/config/account'
import { GetPluginListRequest } from '@/classes/api/req_res/get-plugin-list-request'
import type { PluginInfo } from '@/classes/api/req_res/get-plugin-list-response'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

// 対象アカウントのプラグイン一覧。管理者がアカウント管理画面の行から開く（ADR-0714）。
// プラグインは利用者ごとの実体（$GKILL_HOME/plugins/<利用者>/）なので、
// 一覧も設定も target_user_id で対象を決める。
// 設定の「適用」とは独立したエンティティなので、ServerConfigDialog やスキル一覧と同じく
// 自分で API を呼ぶ。
export function useManagePluginListDialog(options: {
    props: ManagePluginListDialogProps
    emits: ManagePluginListDialogEmits
}) {
    const { props, emits } = options

    const plugin_config_dialog = ref<InstanceType<typeof PluginConfigDialog> | null>(null)
    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)

    const cloned_account: Ref<Account> = ref(new Account())
    const plugins: Ref<Array<PluginInfo>> = ref(new Array<PluginInfo>())
    const is_loading: Ref<boolean> = ref(false)

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("manage-plugin-list-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })

    async function show(account: Account): Promise<void> {
        cloned_account.value = account
        is_show_dialog.value = true
        await reload_plugins()
    }

    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
        plugins.value = new Array<PluginInfo>()
        cloned_account.value = new Account()
    }

    async function reload_plugins(): Promise<void> {
        is_loading.value = true
        try {
            const req = new GetPluginListRequest()
            req.target_user_id = cloned_account.value.user_id
            const res = await props.gkill_api.get_plugin_list(req)
            if (res.errors && res.errors.length !== 0) {
                emits('received_errors', res.errors)
                return
            }
            plugins.value = res.plugins ?? new Array<PluginInfo>()
        } finally {
            is_loading.value = false
        }
    }

    function show_plugin_config_dialog(plugin: PluginInfo): void {
        // 引き当ては rep_name で行う（GetPluginByRepName は manifest の名前でも
        // get_rep_name で申告した名前でも引ける）
        plugin_config_dialog.value?.show(cloned_account.value.user_id, plugin.rep_name)
    }

    return {
        plugin_config_dialog,
        help_dialog,
        cloned_account,
        plugins,
        is_loading,
        is_show_dialog,
        ui,
        show,
        hide,
        reload_plugins,
        show_plugin_config_dialog,
    }
}
