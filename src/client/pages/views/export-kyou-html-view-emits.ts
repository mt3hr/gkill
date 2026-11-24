'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"

export interface ExportKyouHTMLViewEmits {
    (e: 'received_messages', messages: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_close_dialog'): void
    /**
     * 出力中かが変わった。ダイアログはこれを見て、出力中は閉じないようにする
     * （閉じられると、応答が返ってきたときに保存の相手が居ない）。
     */
    (e: 'changed_is_exporting', is_exporting: boolean): void
}
