'use strict'

import type { Kyou } from "@/classes/datas/kyou"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface miKyouViewProps extends KyouViewPropsBase {
    is_readonly_mi_check: boolean
    kyou: Kyou
    height: number | string
    width: number | string
}
