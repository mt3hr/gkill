'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { GkillPropsBase } from "./gkill-props-base"

export interface KyouCountCalendarProps extends GkillPropsBase {
    kyous: Array<Kyou>
    for_mi: boolean
    // 親がv-showで隠しているときfalse。falseの間は件数集計をスキップし、表示時に追いつく
    // (数十万件のkyousを見えないカレンダーへ全件集計しないため)。書き忘れ防止のため必須
    is_active: boolean
}
