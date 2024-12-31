'use strict'

import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"
import type { GkillPropsBase } from "../views/gkill-props-base"

export interface ShareTaskListDialogProps extends GkillPropsBase {
    find_kyou_query: FindKyouQuery
}
