'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"

export interface RepStructContextMenuEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_edit_rep', id: string): void
    (e: 'requested_move_up_rep', id: string): void
    (e: 'requested_move_down_rep', id: string): void
    (e: 'requested_move_rep_to_folder', id: string): void
    (e: 'requested_delete_rep', id: string): void
}