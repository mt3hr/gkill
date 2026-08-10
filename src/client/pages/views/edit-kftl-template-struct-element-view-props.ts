'use strict'

import type { KFTLTemplateElementData } from "@/classes/datas/kftl-template-element-data"
import type { GkillPropsBase } from "./gkill-props-base"

export interface EditKFTLTemplateStructElementViewProps extends GkillPropsBase {
    struct_obj: KFTLTemplateElementData
}

