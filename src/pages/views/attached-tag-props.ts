'use strict'

import type { Tag } from "@/classes/datas/tag"
import type { GkillPropsBase } from "./gkill-props-base"
import type { Kyou } from "@/classes/datas/kyou"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"

export interface AttachedTagProps extends GkillPropsBase {
    last_added_tag: string
    kyou: Kyou
    tag: Tag
    highlight_targets: Array<InfoIdentifier>
}
