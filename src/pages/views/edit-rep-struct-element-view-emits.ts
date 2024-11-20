'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { RepStruct } from "@/classes/datas/config/rep-struct"

export interface EditRepStructElementViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_update_rep_struct', tag_struct: RepStruct): void
    (e: 'requested_close_dialog'): void
}
