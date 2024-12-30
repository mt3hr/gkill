'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import type { MiSortType } from "@/classes/api/find_query/mi-sort-type"
import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"

export interface miSidebarProps extends GkillPropsBase {
    find_kyou_query: FindKyouQuery
}
