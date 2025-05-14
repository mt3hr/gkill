'use strict'

import type { GkillPropsBase } from "./gkill-props-base"

export interface rykvViewProps extends GkillPropsBase {
    app_title_bar_height: Number
    app_content_height: Number
    app_content_width: Number
    is_shared_rykv_view: boolean
    share_title: string
}
