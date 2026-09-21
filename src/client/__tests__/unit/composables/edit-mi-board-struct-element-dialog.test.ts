/**
 * 板の要素編集ダイアログ（use-edit-mi-board-struct-element-dialog）の検証。
 *
 * 8cafc11c で新設。show(struct) が編集対象を載せて開き、hide() と Escape が
 * closeDialogViaHistory 一本（history 駆動の閉じ方）に流れることを固定する。
 * 他の構造要素ダイアログと同じ形で、ここだけ直接 is_show_dialog=false にすると
 * 戻るボタンとダイアログの履歴がずれる。
 */
import { describe, expect, test, vi } from 'vitest'

const history = vi.hoisted(() => ({
    close_dialog_via_history: vi.fn(),
    useDialogHistoryStack: vi.fn(),
}))
const floating = vi.hoisted(() => ({
    name: '' as string,
    options: null as null | { centerMode?: string, onEscape?: () => void },
}))
vi.mock('@/classes/use-dialog-history-stack', () => ({
    useDialogHistoryStack: history.useDialogHistoryStack,
    close_dialog_via_history: history.close_dialog_via_history,
}))
vi.mock('@/classes/use-floating-dialog', () => ({
    useFloatingDialog: vi.fn((name: string, options: { centerMode?: string, onEscape?: () => void }) => {
        floating.name = name
        floating.options = options
        return { floating: true }
    }),
}))

// req_res は GkillAPIRequest を継承する。GkillAPIRequest→GkillAPI→ApplicationConfig→req_res の
// 循環importがあるため、本番同様に gkill-api を先に評価させないと class extends が undefined になる
import '@/classes/api/gkill-api'

import { MiBoardStructElementData } from '@/classes/datas/config/mi-board-struct-element-data'
import { useEditMiBoardStructElementDialog } from '@/classes/use-edit-mi-board-struct-element-dialog'
import type { EditMiBoardStructElementDialogProps } from '@/pages/dialogs/edit-mi-board-struct-element-dialog-props'
import type { EditMiBoardStructElementDialogEmits } from '@/pages/dialogs/edit-mi-board-struct-element-dialog-emits'

function create_dialog() {
    const props = {
        application_config: {},
        gkill_api: {},
        app_content_height: 800,
        app_content_width: 1200,
    } as unknown as EditMiBoardStructElementDialogProps
    const emits = (() => { }) as unknown as EditMiBoardStructElementDialogEmits
    return useEditMiBoardStructElementDialog({ props: props, emits: emits })
}

describe('板の要素編集ダイアログ', () => {
    test('show(struct) は編集対象を載せて開き、履歴スタックに登録している', async () => {
        const dialog = create_dialog()
        expect(dialog.is_show_dialog.value).toBe(false)
        expect(history.useDialogHistoryStack, 'history 駆動の閉じ方に乗っていない').toHaveBeenCalledWith(dialog.is_show_dialog)

        const board = Object.assign(new MiBoardStructElementData(), { id: 'id-board', board_name: 'Inbox', name: 'Inbox' })
        await dialog.show(board)

        expect(dialog.is_show_dialog.value).toBe(true)
        expect(dialog.mi_board_struct.value.id).toBe('id-board')
        expect(dialog.mi_board_struct.value.board_name).toBe('Inbox')
    })

    test('hide() と Escape は closeDialogViaHistory に流れる（直接 false にしない）', async () => {
        const dialog = create_dialog()
        await dialog.show(Object.assign(new MiBoardStructElementData(), { id: 'id-board' }))

        await dialog.hide()
        expect(history.close_dialog_via_history).toHaveBeenCalledWith(dialog.is_show_dialog)

        history.close_dialog_via_history.mockClear()
        expect(floating.name).toBe('edit-mi-board-struct-element-dialog')
        expect(floating.options?.centerMode).toBe('always')
        floating.options?.onEscape?.()
        expect(history.close_dialog_via_history, 'Escape が hide() を通っていない').toHaveBeenCalledWith(dialog.is_show_dialog)
        expect(dialog.ui, 'useFloatingDialog の戻りをそのまま ui として公開する').toEqual({ floating: true })
    })
})
