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
    is_readonly_mi_check: boolean
    enable_context_menu: boolean
    enable_dialog: boolean
    show_content_only: boolean
    show_timeis_plaing_end_button: boolean
}
