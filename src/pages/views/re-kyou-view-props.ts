'use strict'

import type { ReKyou } from "@/classes/datas/re-kyou"
import type { KyouViewPropsBase } from "./kyou-view-props-base"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"

export interface ReKyouViewProps extends KyouViewPropsBase {
    rekyou: ReKyou
    height: number | string
    width: number | string
}
