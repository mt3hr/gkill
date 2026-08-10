'use strict'

export interface ConfirmUnknownMiBoardDialogEmits {
    /** 「保存」を押した。呼び出し元は保存の続きを実行する */
    (e: 'requested_confirm'): void
    /** キャンセル / × を押した。呼び出し元のフォームは開いたままにする */
    (e: 'requested_cancel'): void
}
