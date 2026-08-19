'use strict'

import type { Kyou } from '@/classes/datas/kyou'
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'
import { computed, ref, type Ref } from 'vue'
import type { EditNotificationDialogProps } from '@/pages/dialogs/edit-notification-dialog-props'
import type { KyouDialogEmits } from '@/pages/views/kyou-dialog-emits'
import type { InfoIdentifier } from '@/classes/datas/info-identifier'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useEditNotificationDialog(options: {
    props: EditNotificationDialogProps
    emits: KyouDialogEmits
}) {
    const { props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog, { onClosed: () => emits('closed') })
    const ui = useFloatingDialog("edit-notification-dialog", {
        centerMode: "always",
    })

    const notification_highlight_targets = computed<Array<InfoIdentifier>>(() => {
        const info_identifier = props.notification.generate_info_identifier()
        return [info_identifier]
    })

    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    // クリックはフォーカス移動も伴う
    const crudRelayHandlers = build_kyou_dialog_relay(emits, {
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
    })

    return {
        crudRelayHandlers,
        is_show_dialog,
        ui,
        notification_highlight_targets,
        show,
        hide,
    }
}
