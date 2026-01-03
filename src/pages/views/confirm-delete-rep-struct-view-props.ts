'use strict'

import type { RepStructElementData } from "@/classes/datas/config/rep-struct-element-data"
import type { GkillPropsBase } from "./gkill-props-base"

export interface ConfirmDeleteRepStructViewProps extends GkillPropsBase {
    rep_struct: RepStructElementData
}
