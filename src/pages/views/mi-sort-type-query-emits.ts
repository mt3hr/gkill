'use strict'

import type { MiSortType } from "@/classes/api/find_query/mi-sort-type"

export interface miSortTypeQueryEmits {
    (e: 'request_update_sort_type', sort_type: MiSortType): void
}
