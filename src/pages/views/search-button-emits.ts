'use strict'

export interface SearchButtonEmits {
    (e: 'requested_search'): void
    (e: 'requested_search_with_update_cache'): void
}
