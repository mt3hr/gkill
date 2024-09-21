'use strict'

import type { URLog } from "@/classes/datas/ur-log"
import type { KyouViewPropsBase } from "../views/kyou-view-props-base"

export interface EditURLogDialogProps extends KyouViewPropsBase {
    urlog: URLog
}
