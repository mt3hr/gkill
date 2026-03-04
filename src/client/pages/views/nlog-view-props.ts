'use strict'

import type { Nlog } from "@/classes/datas/nlog"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface NlogViewProps extends KyouViewPropsBase {
    nlog: Nlog
    width: number | string
    height: number | string
}
