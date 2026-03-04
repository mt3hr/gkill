'use strict'

import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"
import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"

export interface FindQueryEditorViewEmits {
    (e: 'requested_apply', find_kyou_query: FindKyouQuery): void
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'inited'): void
    (e: 'updated_query', find_kyou_query: FindKyouQuery): void
    (e: 'requested_close_dialog'): void
}
