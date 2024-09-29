'use strict'

import type { Tag } from "@/classes/datas/tag"
import type { GkillPropsBase } from "./gkill-props-base"
import type { Kyou } from "@/classes/datas/kyou"

export interface TagViewProps extends GkillPropsBase {
    kyou: Kyou
    tag: Tag
    last_added_tag: string
}
