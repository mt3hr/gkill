'use strict'

import type { KFTLTemplateElement } from "@/classes/datas/kftl-template-element"
import type { GkillPropsBase } from "../views/gkill-props-base"

export interface KFTLTemplateDialogProps extends GkillPropsBase {
    templates: Array<KFTLTemplateElement>
}
