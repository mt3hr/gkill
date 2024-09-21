'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { GkillPropsBase } from "./gkill-props-base"

export interface DnoteLocationViewProps extends GkillPropsBase {
    timeis_or_kmemo_kyou: Array<Kyou>
    highlight_targets: Array<Kyou>
    last_added_tag: string
}
