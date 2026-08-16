'use strict'

export interface ConfirmCloseKFTLTabDialogEmits {
    /** 「閉じる」を押した。呼び出し元がタブを閉じる */
    (e: 'requested_confirm'): void
    /** キャンセル / × / Escape。タブはそのまま残す */
    (e: 'requested_cancel'): void
}
