'use strict'

import type { TimeIs } from "@/classes/datas/time-is"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface TimeIsViewProps extends KyouViewPropsBase {
    timeis: TimeIs
    show_timeis_plaing_end_button: boolean
}
