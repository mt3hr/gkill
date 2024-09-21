'use strict'

import type { FindMiQuery } from "@/classes/api/find_query/find-mi-query"
import type { GkillPropsBase } from "./gkill-props-base"
import type { MiSortType } from "@/classes/api/find_query/mi-sort-type"

export interface miSidebarProps extends GkillPropsBase {
    query: FindMiQuery
    board_names: Array<string>
    sort_type: MiSortType
}
