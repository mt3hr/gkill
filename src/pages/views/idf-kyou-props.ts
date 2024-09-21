'use strict'

import type { IDFKyou } from "@/classes/datas/idf-kyou"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface IDFKyouProps extends KyouViewPropsBase {
    idf_kyou: IDFKyou
}
