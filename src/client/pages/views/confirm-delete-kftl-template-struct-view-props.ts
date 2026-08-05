'use strict'

import type { KFTLTemplateStructElementData } from "@/classes/datas/config/kftl-template-struct-element-data"
import type { GkillPropsBase } from "./gkill-props-base"

export interface ConfirmDeleteKFTLTemplateStructViewProps extends GkillPropsBase {
    kftl_template_struct: KFTLTemplateStructElementData
}
