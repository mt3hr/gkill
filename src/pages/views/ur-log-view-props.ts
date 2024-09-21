'use strict'

import type { URLog } from "@/classes/datas/ur-log"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface URLogViewProps extends KyouViewPropsBase {
    urlog: URLog
}
