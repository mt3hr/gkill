'use strict'

import type { UploadFileDialogProps } from '@/pages/dialogs/upload-file-dialog-props'
import { type Ref, ref } from 'vue'
import type { KyouViewEmits } from '@/pages/views/kyou-view-emits';
import type { Kyou } from '@/classes/datas/kyou';
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"
import { build_kyou_dialog_relay } from '@/classes/kyou-view-relay'

export function useUploadFileDialog(options: {
    props: UploadFileDialogProps
    emits: KyouViewEmits
}) {
    const { props: _props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("upload-file-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    // 中身の UploadFileView は KyouViewEmits を全部上げてくる。
    // 手書きで並べていたころは5件しか中継しておらず、
    // タグ・テキスト・通知の追加/削除が呼び出し元へ一切届いていなかった
    const crudRelayHandlers = build_kyou_dialog_relay(emits, {
        // 一覧のクリックはフォーカス移動でもあるので両方上げる
        clicked_kyou: (kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) },
    })
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        is_show_dialog,
        ui,
        crudRelayHandlers,
        show,
        hide,
    }
}
