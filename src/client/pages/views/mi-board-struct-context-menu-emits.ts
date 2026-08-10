'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"

// 板はフォルダ分けも表示名編集もしないので、並べ替えと削除の3項目だけ
export interface MiBoardStructContextMenuEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_move_up_mi_board', id: string): void
    (e: 'requested_move_down_mi_board', id: string): void
    (e: 'requested_delete_mi_board', id: string): void
}
