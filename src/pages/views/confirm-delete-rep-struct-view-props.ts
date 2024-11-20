'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import type { RepStruct } from "@/classes/datas/config/rep-struct"

export interface ConfirmDeleteRepStructViewProps extends GkillPropsBase {
    rep_struct: RepStruct
}
