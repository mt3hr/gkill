'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"

export interface NewBoardNameDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'setted_new_board_name', new_board_name: string): void

}
