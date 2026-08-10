'use strict'

import type { MiBoardStructElementData } from "@/classes/datas/config/mi-board-struct-element-data"
import type { GkillPropsBase } from "./gkill-props-base"

export interface EditMiBoardStructViewProps extends GkillPropsBase {
    mi_board_struct: MiBoardStructElementData
}
