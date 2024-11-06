'use strict'

import type { KFTLTemplateElementData } from "@/classes/datas/kftl-template-element-data"
import type { GkillPropsBase } from "../views/gkill-props-base"

export interface KFTLTemplateDialogProps extends GkillPropsBase {
    templates: Array<KFTLTemplateElementData>
}
