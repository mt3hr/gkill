import { useContextMenuPosition } from '@/classes/use-context-menu-position'
import { i18n } from '@/i18n'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import type { AttachedNotificationContextMenuProps } from '@/pages/views/attached-notification-context-menu-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useAttachedNotificationContextMenu(options: {
    props: AttachedNotificationContextMenuProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    const { is_show, menu_target, open_at } = useContextMenuPosition()

    async function show(e: PointerEvent): Promise<void> {
        open_at(e)
    }

    async function hide(): Promise<void> {
        is_show.value = false
    }

    async function show_edit_notification_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'edit_notification', props.kyou, props.notification)
    }

    async function show_notification_histories_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'notification_histories', props.kyou, props.notification)
    }

    async function copy_id(): Promise<void> {
        navigator.clipboard.writeText(props.notification.id)
        const message = new GkillMessage()
        message.message_code = GkillMessageCodes.copied_notification_id
        message.message = i18n.global.t("COPIED_ID_MESSAGE")
        const messages = new Array<GkillMessage>()
        messages.push(message)
        emits('received_messages', messages)
    }

    async function show_confirm_delete_notification_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'confirm_delete_notification', props.kyou, props.notification)
    }

    return {
        is_show,
        menu_target,
        show,
        hide,
        show_edit_notification_dialog,
        show_notification_histories_dialog,
        copy_id,
        show_confirm_delete_notification_dialog,
    }
}
