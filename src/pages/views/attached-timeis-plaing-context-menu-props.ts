'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import type { Kyou } from "@/classes/datas/kyou"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"

export interface AttachedTimeisPlaingContextMenuProps extends GkillPropsBase {
    timeis_kyou: Kyou
    target_kyou: Kyou
    last_added_tag: string
    highlight_targets: Array<InfoIdentifier>
    enable_context_menu: boolean
    enable_dialog: boolean
}
