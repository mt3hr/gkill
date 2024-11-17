'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { TagStruct } from "@/classes/datas/config/tag-struct"

export interface EditTagStructElementViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_update_tag_struct', tag_struct: TagStruct): void
    (e: 'requested_close_dialog'): void
}
