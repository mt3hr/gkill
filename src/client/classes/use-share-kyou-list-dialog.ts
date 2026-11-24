'use strict'

import type { ShareKyousListDialogEmits } from '@/pages/dialogs/share-kyou-list-dialog-emits'
import type { ShareKyousListDialogProps } from '@/pages/dialogs/share-kyou-list-dialog-props'
import { ref, type Ref } from 'vue'
import { close_dialog_via_history, useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
import { useFloatingDialog } from "@/classes/use-floating-dialog"

export function useShareKyouListDialog(options: {
    props: ShareKyousListDialogProps
    emits: ShareKyousListDialogEmits
}) {
    const { props: _props, emits } = options

    const is_show_dialog: Ref<boolean> = ref(false)
    /**
     * 入口（2つのボタン）か、共有の設定か。
     *
     * 開くたび入口へ戻す。前に開いたときの続きから始まると、
     * 「共有」を押したつもりが設定画面だった、という出方になる
     */
    const mode: Ref<'menu' | 'share'> = ref('menu')
    useDialogHistoryStack(is_show_dialog)
    const ui = useFloatingDialog("share-kyou-list-dialog", {
        centerMode: "always",
        onEscape: () => hide(),
    })
    async function show(): Promise<void> {
        mode.value = 'menu'
        is_show_dialog.value = true
    }

    function show_share_form(): void {
        mode.value = 'share'
    }
    /**
     * 「HTML エクスポート」を選んだ。このダイアログを閉じ、出力ダイアログを開くよう上げる
     * （出力ダイアログはラッパが持っているので、ここからは開けない）
     */
    function request_export_kyou_html(): void {
        void hide()
        emits('requested_export_kyou_html')
    }
    async function hide(): Promise<void> {
        close_dialog_via_history(is_show_dialog)
    }

    return {
        is_show_dialog,
        mode,
        ui,
        show,
        show_share_form,
        request_export_kyou_html,
        hide,
    }
}
