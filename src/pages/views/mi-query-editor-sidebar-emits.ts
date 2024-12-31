'use strict'

import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"
import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"

export interface miQueryEditorSidebarEmits {
    (e: 'request_open_focus_board', board_name: string): void
    (e: 'updated_query_clear', query: FindKyouQuery): void
    (e: 'updated_query', query: FindKyouQuery): void
    (e: 'requested_search', update_cache: boolean): void
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'inited'): void
}
