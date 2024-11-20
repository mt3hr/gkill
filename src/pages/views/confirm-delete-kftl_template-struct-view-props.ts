'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import type { KFTLTemplateStruct } from "@/classes/datas/config/kftl-template-struct"

export interface ConfirmDeleteKFTLTemplateStructViewProps extends GkillPropsBase {
    kftl_template_struct: KFTLTemplateStruct
}
