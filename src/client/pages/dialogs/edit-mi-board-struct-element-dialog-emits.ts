'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { MiBoardStructElementData } from "@/classes/datas/config/mi-board-struct-element-data"

export interface EditMiBoardStructElementDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_update_mi_board_struct', mi_board_struct: MiBoardStructElementData): void
}
