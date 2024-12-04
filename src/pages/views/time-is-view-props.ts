'use strict'

import type { TimeIs } from "@/classes/datas/time-is"
import type { KyouViewPropsBase } from "./kyou-view-props-base"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"

export interface TimeIsViewProps extends KyouViewPropsBase {
    timeis: TimeIs
    show_timeis_plaing_end_button: boolean
    height: number | string
    width: number | string
}
