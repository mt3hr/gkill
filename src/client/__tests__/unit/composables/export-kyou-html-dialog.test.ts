/**
 * useExportKyouHTMLDialog（「HTML で出力」ダイアログ）。
 *
 * 出力の最中に閉じられると、応答が返ってきたときに保存の相手が居ない。×・Escape・戻るのどれでも閉じない。
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import '@/classes/api/gkill-api'
import { useExportKyouHTMLDialog } from '@/classes/use-export-kyou-html-dialog'
import type { ExportKyouHTMLDialogProps } from '@/pages/dialogs/export-kyou-html-dialog-props'
import type { ExportKyouHTMLDialogEmits } from '@/pages/dialogs/export-kyou-html-dialog-emits'

const captured_can_close: Array<(() => boolean) | undefined> = []
vi.mock('@/classes/use-dialog-history-stack', () => ({
    useDialogHistoryStack: (_ref: unknown, opts?: { canClose?: () => boolean }) => {
        captured_can_close.push(opts?.canClose)
    },
    close_dialog_via_history: (flag: { value: boolean }) => {
        flag.value = false
    },
}))

// ダイアログの枠（位置・透過・ドラッグ）はテンプレートへ渡すだけで、composable は中身を読まない
vi.mock('@/classes/use-floating-dialog', () => ({
    useFloatingDialog: () => ({}),
}))

describe('useExportKyouHTMLDialog', () => {
    beforeEach(() => {
        captured_can_close.length = 0
    })

    it('出力中（is_exporting=true）は hide() も canClose も閉じるのを止める', async () => {
        const dialog = useExportKyouHTMLDialog({
            props: {} as ExportKyouHTMLDialogProps,
            emits: (() => {}) as unknown as ExportKyouHTMLDialogEmits,
        })
        const canClose = captured_can_close[0]
        expect(canClose).toBeTypeOf('function')

        await dialog.show()
        expect(dialog.is_show_dialog.value).toBe(true)
        expect(canClose!()).toBe(true)

        dialog.set_is_exporting(true)
        expect(canClose!()).toBe(false)
        await dialog.hide()
        expect(dialog.is_show_dialog.value, '出力中は閉じない').toBe(true)

        dialog.set_is_exporting(false)
        expect(canClose!()).toBe(true)
        await dialog.hide()
        expect(dialog.is_show_dialog.value, '終了後は閉じられる').toBe(false)
    })
})
