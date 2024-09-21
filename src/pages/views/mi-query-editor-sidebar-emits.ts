'use strict'

import type { FindMiQuery } from "@/classes/api/find_query/find-mi-query"

export interface miQueryEditorSidebarEmits {
    (e: 'updated_query', query: FindMiQuery): void
    (e: 'request_search'): void
    (e: 'request_open_focus_board', board_name: string): void
}
