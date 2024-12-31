'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { GkillPropsBase } from "./gkill-props-base"
import { Text } from "@/classes/datas/text"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"

export interface TextViewProps extends GkillPropsBase {
    text: Text
    last_added_tag: string
    kyou: Kyou
    highlight_targets: Array<InfoIdentifier>
    enable_context_menu: boolean
    enable_dialog: boolean
}
