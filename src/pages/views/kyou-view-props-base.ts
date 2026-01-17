'use strict'

import type { InfoIdentifier } from "@/classes/datas/info-identifier"
import type { Kyou } from "@/classes/datas/kyou"
import type { GkillPropsBase } from "./gkill-props-base"

export interface KyouViewPropsBase extends GkillPropsBase {
    kyou: Kyou
    highlight_targets: Array<InfoIdentifier>
    last_added_tag: string
    enable_context_menu: boolean
    enable_dialog: boolean
    draggable?: boolean
}
