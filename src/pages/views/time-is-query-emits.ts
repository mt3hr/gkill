'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"

export interface TimeIsQueryEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'request_clear_timeis_query'): void
    (e: 'request_update_use_timeis_query', use_timeis_query: boolean): void
    (e: 'request_update_timeis_keywords', timeis_keyword: string): void
    (e: 'request_update_and_search_timeis_word', and_search_timeis_word: boolean): void
    (e: 'request_update_and_search_timeis_tags', and_search_timeis_tags: boolean): void
    (e: 'request_update_checked_timeis_tags', checked_tags: Array<string>, is_by_user: boolean): void
    (e: 'inited'): void
}
