'use strict';

export interface SidebarHeaderEmits {
    (e: 'request_search'): void
    (e: 'request_clear_find_query'): void
}