'use strict'

import type { ReKyou } from "@/classes/datas/re-kyou"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface ReKyouViewProps extends KyouViewPropsBase {
    rekyou: ReKyou
}
