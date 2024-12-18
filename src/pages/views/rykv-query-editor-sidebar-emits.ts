'use strict'

import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"

export interface rykvQueryEditorSidebarEmits {
    (e: 'updated_query_clear', query: FindKyouQuery): void
    (e: 'updated_query', query: FindKyouQuery): void
    (e: 'requested_search'): void
}
