'use strict'

import type { GkillPropsBase } from "./gkill-props-base"

export interface GPSLogMapProps extends GkillPropsBase {
    start_time: Date
    end_time: Date
    marker_time: Date
    app_content_height: number
}
