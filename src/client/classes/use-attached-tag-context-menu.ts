import { useContextMenuPosition } from '@/classes/use-context-menu-position'
import { i18n } from '@/i18n'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import type { AttachedTagContextMenuProps } from '@/pages/views/attached-tag-context-menu-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useAttachedTagContextMenu(options: {
    props: AttachedTagContextMenuProps,
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

    async function show_edit_tag_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'edit_tag', props.kyou, props.tag)
    }

    async function show_tag_histories_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'tag_histories', props.kyou, props.tag)
    }

    async function copy_id(): Promise<void> {
        navigator.clipboard.writeText(props.tag.id)
        const message = new GkillMessage()
        message.message_code = GkillMessageCodes.copied_tag_id
        message.message = i18n.global.t("COPIED_ID_MESSAGE")
        const messages = new Array<GkillMessage>()
        messages.push(message)
        emits('received_messages', messages)
    }

    async function show_confirm_delete_tag_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'confirm_delete_tag', props.kyou, props.tag)
    }

    return {
        is_show,
        menu_target,
        show,
        hide,
        show_edit_tag_dialog,
        show_tag_histories_dialog,
        copy_id,
        show_confirm_delete_tag_dialog,
    }
}
