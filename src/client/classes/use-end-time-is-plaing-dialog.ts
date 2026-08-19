'use strict'

import { type Ref, ref } from 'vue'
import type { EndTimeIsPlaingDialogProps } from '@/pages/dialogs/end-time-is-plaing-dialog-props'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits'
import type { Kyou } from '@/classes/datas/kyou'
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useEndTimeIsPlaingDialog(options: {
    props: EndTimeIsPlaingDialogProps
    emits: KyouViewEmits
}) {
    const { props: _props, emits } = options

    // クリックはフォーカス移動も伴う
    const crudRelayHandlers = build_kyou_dialog_relay(emits, {
        'clicked_kyou': (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
    })
    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("end-time-is-plaing-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
        // 編集できる欄が1つも無い（タイトルも日時も readonly の見せかけ入力）ので
        // フォーカスは当てない。当てるとスマートフォンで無意味にキーボードが出る
        autofocus: false,
    })
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        crudRelayHandlers,
        is_show_dialog,
        ui,
        show,
        hide,
    }
}
