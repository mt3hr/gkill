'use strict'

import type { KC } from "@/classes/datas/kc"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface KCViewProps extends KyouViewPropsBase {
    kc: KC
    height: number | string
    width: number | string
}
