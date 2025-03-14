'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import type { RepStruct } from "@/classes/datas/config/rep-struct"

export interface EditRepStructElementViewProps extends GkillPropsBase {
    struct_obj: RepStruct
    folder_name: string
}
