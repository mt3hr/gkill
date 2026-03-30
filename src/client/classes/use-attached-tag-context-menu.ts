import { i18n } from '@/i18n'
import { computed, type Ref, ref } from 'vue'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import type { AttachedTagContextMenuProps } from '@/pages/views/attached-tag-context-menu-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'

export function useAttachedTagContextMenu(options: {
    props: AttachedTagContextMenuProps,
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
        position_x,
        position_y,
        context_menu_style,
        show,
        hide,
        show_edit_tag_dialog,
        show_tag_histories_dialog,
        copy_id,
        show_confirm_delete_tag_dialog,
    }
}
