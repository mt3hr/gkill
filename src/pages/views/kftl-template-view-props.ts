'use strict'

import type { KFTLTemplateElementData } from "@/classes/datas/kftl-template-element-data"
import type { GkillPropsBase } from "./gkill-props-base"

export interface KFTLTemplateViewProps extends GkillPropsBase {
    template: KFTLTemplateElementData
}
