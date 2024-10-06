'use strict'

import type { Tag } from "@/classes/datas/tag"
import type { GkillPropsBase } from "./gkill-props-base"
import type { Kyou } from "@/classes/datas/kyou"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"

export interface TagHistoriesViewProps extends GkillPropsBase {
    tag: Tag
    kyou: Kyou
    last_added_tag: string
    highlight_targets: Array<InfoIdentifier>
}
