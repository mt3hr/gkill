'use strict'

import type { TimeIs } from "@/classes/datas/time-is"
import type { GkillPropsBase } from "./gkill-props-base"
import type { Kyou } from "@/classes/datas/kyou"

export interface AttachedTimeIsPlaingProps extends GkillPropsBase {
    timeis: TimeIs
    kyou: Kyou
    last_added_tag: string
}
