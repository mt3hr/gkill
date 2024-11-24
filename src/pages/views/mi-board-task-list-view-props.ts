'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import type { Kyou } from "@/classes/datas/kyou"
import type { FindKyouQuery } from "@/classes/api/find_query/find-kyou-query"

export interface miBoardTaskListViewProps extends GkillPropsBase {
    app_content_height: Number
    app_content_width: Number
    query: FindKyouQuery
    matched_kyous: Array<Kyou>
    last_added_tag: string
    is_show_close_button: boolean
}
