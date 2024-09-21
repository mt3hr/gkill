'use strict'

export interface miSidebarHeaderEmits {
    (e: 'request_search'): void
    (e: 'request_clear_find_query'): void
}
