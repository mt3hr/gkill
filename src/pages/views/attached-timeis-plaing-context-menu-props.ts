'use strict'

import type { TimeIs } from "@/classes/datas/time-is"
import type { GkillPropsBase } from "./gkill-props-base"
import type { Kyou } from "@/classes/datas/kyou"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"

export interface AttachedTimeisPlaingContextMenuProps extends GkillPropsBase {
    timeis: TimeIs
    kyou: Kyou
    last_added_tag: string
    highlight_targets: Array<InfoIdentifier>
}
