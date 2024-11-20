'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import type { RepTypeStruct } from "@/classes/datas/config/rep-type-struct"

export interface ConfirmDeleteRepTypeStructViewProps extends GkillPropsBase {
    rep_type_struct: RepTypeStruct
}
