'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import type { Kyou } from "@/classes/datas/kyou"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"
export interface AttachedTimeIsPlaingProps extends GkillPropsBase {
    timeis_kyou: Kyou
    kyou: Kyou
    highlight_targets: Array<InfoIdentifier>
    enable_context_menu: boolean
    enable_dialog: boolean
}
