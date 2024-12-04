'use strict'

import type { IDFKyou } from "@/classes/datas/idf-kyou"
import type { KyouViewPropsBase } from "./kyou-view-props-base"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"

export interface IDFKyouProps extends KyouViewPropsBase {
    idf_kyou: IDFKyou
    height: number | string
    width: number | string
}
