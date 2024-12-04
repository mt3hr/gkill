'use strict'

import type { Nlog } from "@/classes/datas/nlog"
import type { KyouViewPropsBase } from "./kyou-view-props-base"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"

export interface NlogViewProps extends KyouViewPropsBase {
    nlog: Nlog
    width: number | string
    height: number | string
}
