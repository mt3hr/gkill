import { computed, type Ref, ref } from 'vue'
import type { InfoIdentifier } from '@/classes/datas/info-identifier'
import type { Kyou } from '@/classes/datas/kyou'
import type { NotificationHistoriesDialogProps } from '@/pages/dialogs/notification-histories-dialog-props'
import type { KyouDialogEmits } from '@/pages/views/kyou-dialog-emits'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'

export function useNotificationHistoriesDialog(options: {
    props: NotificationHistoriesDialogProps,
    emits: KyouDialogEmits,
}) {
    const { props, emits } = options

    // ── Dialog state ──
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog, { onClosed: () => emits('closed') })
    const ui = useFloatingDialog("notification-histories-dialog", {
        centerMode: "always",
    })

    const show_kyou: Ref<boolean> = ref(false)

    // ── Computed ──
    const notification_highlight_targets = computed<Array<InfoIdentifier>>(() => {
        const info_identifier = props.notification.generate_info_identifier()
        return [info_identifier]
    })

    // ── CRUD relay handlers ──
    const crudRelayHandlers = build_kyou_dialog_relay(emits, {
        // クリックはフォーカス移動も伴う
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
    })

    // ── Methods ──
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    // ── Return ──
    return {
        // State
        is_show_dialog,
        ui,
        show_kyou,
        notification_highlight_targets,

        // Methods
        show,
        hide,

        // Event relay objects
        crudRelayHandlers,
    }
}
