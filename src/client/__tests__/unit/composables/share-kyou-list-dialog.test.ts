/**
 * useShareKyouListDialog（共有ダイアログ）。渡し方の入口で「HTML エクスポート」と「画面を共有」を選ばせる。
 */
import { describe, it, expect, vi } from 'vitest'
import '@/classes/api/gkill-api'
import { useShareKyouListDialog } from '@/classes/use-share-kyou-list-dialog'
import type { ShareKyousListDialogProps } from '@/pages/dialogs/share-kyou-list-dialog-props'
import type { ShareKyousListDialogEmits } from '@/pages/dialogs/share-kyou-list-dialog-emits'

vi.mock('@/classes/use-dialog-history-stack', () => ({
    useDialogHistoryStack: () => {},
    close_dialog_via_history: (flag: { value: boolean }) => {
        flag.value = false
    },
}))

// ダイアログの枠（位置・透過・ドラッグ）はテンプレートへ渡すだけで、composable は中身を読まない
vi.mock('@/classes/use-floating-dialog', () => ({
    useFloatingDialog: () => ({}),
}))

describe('useShareKyouListDialog', () => {
    it('show() のたびに mode が入口（menu）へ戻る', async () => {
        const dialog = useShareKyouListDialog({
            props: {} as ShareKyousListDialogProps,
            emits: (() => {}) as unknown as ShareKyousListDialogEmits,
        })
        await dialog.show()
        expect(dialog.mode.value).toBe('menu')

        dialog.show_share_form()
        expect(dialog.mode.value).toBe('share')

        await dialog.hide()
        expect(dialog.is_show_dialog.value).toBe(false)

        await dialog.show()
        expect(dialog.mode.value, '再表示時は menu に戻る').toBe('menu')
    })

    // 出力ダイアログはラッパ（rykv-view / mi-view）が持っているので、ここは閉じて依頼を上げるだけ。
    // 上げ忘れると「HTML エクスポート」を押してもダイアログが閉じるだけで何も起きない
    it('「HTML エクスポート」は自分を閉じて requested_export_kyou_html を上げる', async () => {
        const emitted: Array<string> = []
        const dialog = useShareKyouListDialog({
            props: {} as ShareKyousListDialogProps,
            emits: ((event: string) => { emitted.push(event) }) as unknown as ShareKyousListDialogEmits,
        })
        await dialog.show()
        dialog.request_export_kyou_html()
        expect(dialog.is_show_dialog.value, '共有ダイアログが開いたまま').toBe(false)
        expect(emitted).toEqual(['requested_export_kyou_html'])
    })
})
