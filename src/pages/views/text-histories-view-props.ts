'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { GkillPropsBase } from "./gkill-props-base"
import { Text } from "@/classes/datas/text"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"

export interface TextHistoriesViewProps extends GkillPropsBase {
    text: Text
    kyou: Kyou
    last_added_tag: string
    highlight_targets: Array<InfoIdentifier>
    enable_context_menu: boolean
    enable_dialog: boolean
}
