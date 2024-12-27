'use strict'

import type { FindKyouQuery } from '@/classes/api/find_query/find-kyou-query'
import type { Kyou } from '@/classes/datas/kyou'
import type { GkillPropsBase } from './gkill-props-base'

export interface KyouListViewProps extends GkillPropsBase {
    is_focused_list: boolean
    last_added_tag: string
    query: FindKyouQuery
    matched_kyous: Array<Kyou>
    list_height: Number
    kyou_height: Number
    width: Number
    show_footer: boolean
    show_checkbox: boolean
    closable: boolean
}
