'use strict'

import type { RepTypeStructElementData } from "@/classes/datas/config/rep-type-struct-element-data"
import type { GkillPropsBase } from "./gkill-props-base"

export interface EditRepTypeStructElementViewProps extends GkillPropsBase {
    struct_obj: RepTypeStructElementData
}

