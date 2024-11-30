'use strict'

export interface SidebarHeaderEmits {
    (e: 'requested_search'): void
    (e: 'requested_clear_find_query'): void
}
