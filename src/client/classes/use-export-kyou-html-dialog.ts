'use strict'

import type { ExportKyouHTMLDialogEmits } from '@/pages/dialogs/export-kyou-html-dialog-emits'
import type { ExportKyouHTMLDialogProps } from '@/pages/dialogs/export-kyou-html-dialog-props'
import { ref, type Ref } from 'vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from '@/classes/use-floating-dialog'

export function useExportKyouHTMLDialog(options: {
    props: ExportKyouHTMLDialogProps
    emits: ExportKyouHTMLDialogEmits
}) {
    const { props: _props, emits: _emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    /**
     * 出力中か。**立っている間は閉じられない。**
     *
     * 出力の最中に閉じられると、応答が返ってきたときに保存の相手が居ない。
     * 中の画面（export-kyou-html-view）が出力の開始と終了で書き、
     * × ・ Esc ・ キャンセルはこれを見る。キャンセルを押すと中断されて false に戻るので、
     * そのあとなら閉じられる。
     */
    const is_exporting: Ref<boolean> = ref(false)

    // 出力中はブラウザバックでも Escape でも閉じない（× は hide() が見る）
    useDialogHistoryStack(is_show_dialog, { canClose: () => !is_exporting.value })
    const ui = useFloatingDialog("export-kyou-html-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(): Promise<void> {
        is_show_dialog.value = true
    }
    async function hide(): Promise<void> {
        if (is_exporting.value) {
            return
        }
        close_dialog_via_history(is_show_dialog)
    }

    /** 中の画面から「出力中かが変わった」を受ける */
    function set_is_exporting(value: boolean): void {
        is_exporting.value = value
    }

    return {
        is_show_dialog,
        is_exporting,
        ui,
        show,
        hide,
        set_is_exporting,
    }
}
