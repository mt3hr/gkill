'use strict'

import type { TagStructElementData } from "@/classes/datas/config/tag-struct-element-data"
import type { GkillPropsBase } from "./gkill-props-base"

export interface EditTagStructViewProps extends GkillPropsBase {
    tag_struct: TagStructElementData
}
