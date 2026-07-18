'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"

export interface KFTLTemplateStructContextMenuEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_edit_kftl_template', id: string): void
    (e: 'requested_move_up_kftl_template', id: string): void
    (e: 'requested_move_down_kftl_template', id: string): void
    (e: 'requested_move_kftl_template_to_folder', id: string): void
    (e: 'requested_delete_kftl_template', id: string): void
}
