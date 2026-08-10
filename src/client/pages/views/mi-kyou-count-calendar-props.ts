'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { MiSortType } from "@/classes/api/find_query/mi-sort-type"
import type { GkillPropsBase } from "./gkill-props-base"

export interface MiKyouCountCalendarProps extends GkillPropsBase {
    kyous: Array<Kyou>
    mi_sort_type: MiSortType
    // 親がv-showで隠しているときfalse。falseの間は件数集計をスキップし、表示時に追いつく
    // (数十万件のkyousを見えないカレンダーへ全件集計しないため)。書き忘れ防止のため必須
    is_active: boolean
}
