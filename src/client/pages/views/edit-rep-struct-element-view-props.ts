'use strict'

import type { RepStructElementData } from "@/classes/datas/config/rep-struct-element-data"
import type { GkillPropsBase } from "./gkill-props-base"

export interface EditRepStructElementViewProps extends GkillPropsBase {
    struct_obj: RepStructElementData
    folder_name: string
}
