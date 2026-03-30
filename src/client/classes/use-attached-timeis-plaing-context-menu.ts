'use strict'

import { computed, type Ref, ref, watch } from 'vue'
import { i18n } from '@/i18n'
import type { AttachedTimeisPlaingContextMenuProps } from '@/pages/views/attached-timeis-plaing-context-menu-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import { GkillMessage } from '@/classes/api/gkill-message'
import { GkillMessageCodes } from '@/classes/api/message/gkill_message'

export function useAttachedTimeisPlaingContextMenu(options: { props: AttachedTimeisPlaingContextMenuProps, emits: KyouViewEmits }) {
    const { props, emits } = options

    const cloned_timeis_kyou: Ref<Kyou> = ref(props.timeis_kyou.clone())

    watch(() => props.timeis_kyou, () => {
        reload_cloned_timeis_kyou()
    })

    reload_cloned_timeis_kyou()

    const is_show: Ref<boolean> = ref(false)
    const position_x: Ref<number> = ref(0)
    const position_y: Ref<number> = ref(0)
    const context_menu_style = computed(() => `{ position: absolute; left: ${Math.min(document.defaultView!.innerWidth - 130, position_x.value.valueOf())}px; top: ${Math.min(Math.max(50, document.defaultView!.innerHeight - ( + 8 + (48 * 4))), position_y.value.valueOf())}px; }`)

    function reload_cloned_timeis_kyou(): void {
        cloned_timeis_kyou.value = props.timeis_kyou.clone()
        cloned_timeis_kyou.value.load_typed_datas()
        cloned_timeis_kyou.value.load_attached_histories()
    }

    async function show(e: PointerEvent): Promise<void> {
        position_x.value = e.clientX
        position_y.value = e.clientY
        is_show.value = true
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
        position_x,
        position_y,
        context_menu_style,
        show,
        hide,
        show_edit_timeis_dialog,
        show_timeis_histories_dialog,
        copy_id,
        show_confirm_delete_timeis_dialog,
    }
}
