'use strict'

import type { KFTLTemplateStructElementData } from "@/classes/datas/config/kftl-template-struct-element-data"
import type { GkillPropsBase } from "./gkill-props-base"

export interface EditKFTLTemplateStructElementViewProps extends GkillPropsBase {
    struct_obj: KFTLTemplateStructElementData
}

