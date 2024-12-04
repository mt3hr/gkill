'use strict'

import type { URLog } from "@/classes/datas/ur-log"
import type { KyouViewPropsBase } from "./kyou-view-props-base"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"

export interface URLogViewProps extends KyouViewPropsBase {
    urlog: URLog
    height: number | string
    width: number | string
}
