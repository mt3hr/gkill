import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import type { AttachedNotificationContextMenuProps } from '@/pages/views/attached-notification-context-menu-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useAttachedNotificationContextMenu(options: {
    props: AttachedNotificationContextMenuProps,
    emits: KyouViewEmits,
}) {
    const { props, emits } = options

    const is_show: Ref<boolean> = ref(false)
    const position_x: Ref<number> = ref(0)
    const position_y: Ref<number> = ref(0)
    const context_menu_style = computed(() => `{ position: absolute; left: ${Math.min(document.defaultView!.innerWidth - 130, position_x.value.valueOf())}px; top: ${Math.min(Math.max(50, document.defaultView!.innerHeight - ( + 8 + (48 * 4))), position_y.value.valueOf())}px; }`)

    async function show(e: PointerEvent): Promise<void> {
        position_x.value = e.clientX
        position_y.value = e.clientY
        is_show.value = true
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
        position_x,
        position_y,
        context_menu_style,
        show,
        hide,
        show_edit_notification_dialog,
        show_notification_histories_dialog,
        copy_id,
        show_confirm_delete_notification_dialog,
    }
}
