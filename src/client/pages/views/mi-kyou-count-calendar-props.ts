'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { MiSortType } from "@/classes/api/find_query/mi-sort-type"
import type { GkillPropsBase } from "./gkill-props-base"

export interface MiKyouCountCalendarProps extends GkillPropsBase {
    kyous: Array<Kyou>
    mi_sort_type: MiSortType
}
