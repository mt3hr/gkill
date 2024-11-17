'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import type { TagStruct } from "@/classes/datas/config/tag-struct"

export interface EditTagStructElementViewProps extends GkillPropsBase {
    struct_obj: TagStruct
}
