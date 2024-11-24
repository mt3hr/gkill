'use strict'

import type { InfoIdentifier } from "@/classes/datas/info-identifier"
import type { GkillPropsBase } from "./gkill-props-base"

export interface miViewProps extends GkillPropsBase {
    app_content_height: Number
    app_content_width: Number
    highlight_targets: Array<InfoIdentifier>
    last_added_tag: string
}
