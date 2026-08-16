'use strict'

export interface ConfirmUnknownTagDialogEmits {
    /** 「保存」を押した。呼び出し元は保存の続きを実行する */
    (e: 'requested_confirm'): void
    /** キャンセル / × を押した。呼び出し元のフォームは開いたままにする */
    (e: 'requested_cancel'): void
    /**
     * 閉じた。保存・キャンセル・×・Escape・ブラウザバックのどれでも1回だけ上がる。
     *
     * 「確認が開いているか」を呼び出し元が持ちたいときはこれで倒すこと。
     * `unknown_tags` の空判定で代用すると、ブラウザバックでは空にならないので
     * 開きっぱなし扱いのままになる（KFTLのタブ操作ロックが固まる）
     */
    (e: 'closed'): void
}
