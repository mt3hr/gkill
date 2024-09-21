'use strict'

import type { TimeIs } from "@/classes/datas/time-is"
import type { KyouViewPropsBase } from "../views/kyou-view-props-base"

export interface EditTimeIsDialogProps extends KyouViewPropsBase {
    timeis: TimeIs
}
