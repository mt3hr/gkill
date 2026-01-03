'use strict'

import type { RepTypeStructElementData } from "@/classes/datas/config/rep-type-struct-element-data"
import type { GkillPropsBase } from "./gkill-props-base"

export interface EditRepTypeStructViewProps extends GkillPropsBase {
    rep_type_struct: RepTypeStructElementData
}

