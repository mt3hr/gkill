'use strict'

import type { MiReKyou } from "@/classes/datas/mi-re-kyou"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface MiReKyouViewProps extends KyouViewPropsBase {
    mirekyou: MiReKyou
    is_readonly_mi_check: boolean
    height: number | string
    width: number | string
}
