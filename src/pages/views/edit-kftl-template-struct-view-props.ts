'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import { KFTLTemplateStruct } from "@/classes/datas/config/kftl-template-struct"

export interface EditKFTLTemplateStructViewProps extends GkillPropsBase {
    kftl_template_struct: Array<KFTLTemplateStruct>
}
