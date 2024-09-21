'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { GkillPropsBase } from "./gkill-props-base"

export interface KyouCountCalendarProps extends GkillPropsBase {
    kyous: Array<Kyou>
}
