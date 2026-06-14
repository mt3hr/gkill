'use strict'

import type { FindKyouQuery } from "../../classes/api/find_query/find-kyou-query"
import type { GkillError } from "../../classes/api/gkill-error"
import type { GkillMessage } from "../../classes/api/gkill-message"

export interface MiFindQueryEditorDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_close_dialog'): void
    (e: 'requested_apply', query: FindKyouQuery): void
}
