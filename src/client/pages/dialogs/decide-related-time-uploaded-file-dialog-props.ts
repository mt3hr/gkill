'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { GkillPropsBase } from "../views/gkill-props-base"

export interface DecideRelatedTimeUploadedFileDialogProps extends GkillPropsBase {
    app_content_height: number
    app_content_width: number
    uploaded_kyous: Array<Kyou>
}
