'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import { RepTypeStruct } from "@/classes/datas/config/rep-type-struct"

export interface EditRepTypeStructViewProps extends GkillPropsBase {
    rep_type_struct: Array<RepTypeStruct>
}

