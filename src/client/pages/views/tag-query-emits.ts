'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"

export interface TagQueryEmits {
    (e: 'request_clear_tag_query'): void
    (e: 'request_update_checked_tags', checked_tags: Array<string>, is_by_user: boolean): void
    (e: 'request_update_and_search_tags', and_search_tags: boolean): void
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'inited'): void
}
