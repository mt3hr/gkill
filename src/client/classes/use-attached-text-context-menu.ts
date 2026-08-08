import { useContextMenuPosition } from '@/classes/use-context-menu-position'
import { i18n } from '@/i18n'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import type { AttachedTextContextMenuProps } from '@/pages/views/attached-text-context-menu-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useAttachedTextContextMenu(options: {
    props: AttachedTextContextMenuProps,
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

    async function show_edit_text_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'edit_text', props.kyou, props.text)
    }

    async function show_text_histories_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'text_histories', props.kyou, props.text)
    }

    async function copy_id(): Promise<void> {
        navigator.clipboard.writeText(props.text.id)
        const message = new GkillMessage()
        message.message_code = GkillMessageCodes.copied_text_id
        message.message = i18n.global.t("COPIED_ID_MESSAGE")
        const messages = new Array<GkillMessage>()
        messages.push(message)
        emits('received_messages', messages)
    }

    async function show_confirm_delete_text_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'confirm_delete_text', props.kyou, props.text)
    }

    return {
        is_show,
        menu_target,
        show,
        hide,
        show_edit_text_dialog,
        show_text_histories_dialog,
        copy_id,
        show_confirm_delete_text_dialog,
    }
}
