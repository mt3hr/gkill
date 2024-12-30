'use strict'

import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"

export interface miQueryEditorSidebarEmits {
    (e: 'request_open_focus_board', board_name: string): void
    (e: 'updated_query_clear', query: FindKyouQuery): void
    (e: 'updated_query', query: FindKyouQuery): void
    (e: 'requested_search', update_cache: boolean): void
    (e: 'inited'): void
}
