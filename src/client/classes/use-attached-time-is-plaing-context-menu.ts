'use strict'

import { ref, type Ref, watch } from 'vue'
import { useContextMenuPosition } from '@/classes/use-context-menu-position'
import { i18n } from '@/i18n'
import type { AttachedTimeIsPlaingContextMenuProps } from '@/pages/views/attached-time-is-plaing-context-menu-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'
import { copy_kyou_content } from '@/classes/kyou-content-text'

export function useAttachedTimeIsPlaingContextMenu(options: { props: AttachedTimeIsPlaingContextMenuProps, emits: KyouViewEmits }) {
    const { props, emits } = options

    const cloned_timeis_kyou: Ref<Kyou> = ref(props.timeis_kyou.clone())

    watch(() => props.timeis_kyou, () => {
        reload_cloned_timeis_kyou()
    })

    reload_cloned_timeis_kyou()

    const { is_show, menu_target, open_at } = useContextMenuPosition()

    function reload_cloned_timeis_kyou(): void {
        cloned_timeis_kyou.value = props.timeis_kyou.clone()
        cloned_timeis_kyou.value.load_typed_datas()
        cloned_timeis_kyou.value.load_attached_histories()
    }

    async function show(e: PointerEvent): Promise<void> {
        open_at(e)
    }

    async function hide(): Promise<void> {
        is_show.value = false
    }

    async function show_edit_timeis_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'edit_timeis', cloned_timeis_kyou.value)
    }

    async function show_timeis_histories_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'kyou_histories', cloned_timeis_kyou.value)
    }

    async function copy_content(): Promise<void> {
        const res = await copy_kyou_content(cloned_timeis_kyou.value, props.gkill_api)
        if (res.errors.length !== 0) {
            emits('received_errors', res.errors)
            return
        }
        if (res.messages.length !== 0) {
            emits('received_messages', res.messages)
        }
    }

    async function copy_id(): Promise<void> {
        navigator.clipboard.writeText(props.timeis_kyou.id)
        const message = new GkillMessage()
        message.message_code = GkillMessageCodes.copied_timeis_id
        message.message = i18n.global.t("COPIED_ID_MESSAGE")
        const messages = new Array<GkillMessage>()
        messages.push(message)
        emits('received_messages', messages)
    }

    async function show_confirm_delete_timeis_dialog(): Promise<void> {
        emits('requested_open_rykv_dialog', 'confirm_delete_kyou', cloned_timeis_kyou.value)
    }

    return {
        cloned_timeis_kyou,
        is_show,
        menu_target,
        show,
        hide,
        show_edit_timeis_dialog,
        show_timeis_histories_dialog,
        copy_content,
        copy_id,
        show_confirm_delete_timeis_dialog,
    }
}
