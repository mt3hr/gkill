'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"

export interface MiShareFooterEmits {
    (e: 'request_open_share_kyou_dialog'): void
    (e: 'request_open_manage_share_kyou_dialog'): void
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
}
