'use strict'

import type { GkillPropsBase } from "./gkill-props-base"

export interface miSharedTaskViewProps extends GkillPropsBase {
    app_title_bar_height: number
    app_content_height: number
    app_content_width: number
    share_id: string
}
