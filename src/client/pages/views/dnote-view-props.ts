'use strict'

import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"
import type { GkillPropsBase } from "./gkill-props-base"
import type { Kyou } from "@/classes/datas/kyou"

export interface DnoteViewProps extends GkillPropsBase {
    query: FindKyouQuery
    checked_kyous: Array<Kyou>
    app_content_height: Number
    app_content_width: Number
    last_added_tag: string
    editable: boolean
}
