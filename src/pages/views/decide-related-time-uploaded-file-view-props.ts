'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { GkillPropsBase } from "./gkill-props-base"

export interface DecideRelatedTimeUploadedFileViewProps extends GkillPropsBase {
    app_content_height: Number
    app_content_width: Number
    uploaded_kyous: Array<Kyou>
    last_added_tag: string
}
