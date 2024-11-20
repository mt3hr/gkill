'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { RepTypeStruct } from "@/classes/datas/config/rep-type-struct"

export interface EditRepTypeStructElementDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_update_rep_type_struct', rep_type_struct: RepTypeStruct): void
}
