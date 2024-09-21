'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { GkillPropsBase } from "./gkill-props-base"

export interface DecideRelatedTimeUploadedFileViewProps extends GkillPropsBase {
    uploaded_kyous: Array<Kyou>
    last_added_tag: string
}
