'use strict'

export interface ConfirmSaveDuplicatedSharedDataDialogEmits {
    /** 「それでも保存する」を押した。呼び出し元が保存し直す */
    (e: 'requested_save'): void
    /** キャンセル / × / Escape。保存しないで終わる */
    (e: 'requested_cancel'): void
}
