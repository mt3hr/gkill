'use strict'

import type { TagStructElementData } from "@/classes/datas/config/tag-struct-element-data"
import type { GkillPropsBase } from "./gkill-props-base"

export interface EditTagStructElementViewProps extends GkillPropsBase {
    struct_obj: TagStructElementData
}
