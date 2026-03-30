'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { GkillPropsBase } from "./gkill-props-base"

export interface DecideRelatedTimeUploadedFileViewProps extends GkillPropsBase {
    app_content_height: number
    app_content_width: number
    uploaded_kyous: Array<Kyou>
}
