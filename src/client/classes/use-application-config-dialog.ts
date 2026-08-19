'use strict'

import { type Ref, ref, nextTick } from 'vue'
import type { ApplicationConfigDialogProps } from '@/pages/dialogs/application-config-dialog-props'
import type { ApplicationConfigDialogEmits } from '@/pages/dialogs/application-config-dialog-emits'
import ApplicationConfigView from '@/pages/views/application-config-view.vue'
import HelpDialog from '@/pages/dialogs/help-dialog.vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useApplicationConfigDialog(options: {
    props: ApplicationConfigDialogProps
    emits: ApplicationConfigDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const application_config_view = ref<InstanceType<typeof ApplicationConfigView> | null>(null);
    const help_dialog = ref<InstanceType<typeof HelpDialog> | null>(null)
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("application-config-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(): Promise<void> {
        is_show_dialog.value = true
        await nextTick()
        application_config_view.value?.reload_cloned_application_config()
    }
    async function hide(): Promise<void> {
        // ×・Escape・キャンセルのどれで閉じても、「適用」していない変更は破棄する。
        // ロケールとダークテーマは選ばせるために即時プレビューしているので、明示的に戻す必要がある
        application_config_view.value?.cancel_pending_changes()
        close_dialog_via_history(is_show_dialog)
    }

    return {
        application_config_view,
        help_dialog,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
