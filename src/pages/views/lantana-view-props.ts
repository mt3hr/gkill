'use strict'

import type { Lantana } from "@/classes/datas/lantana"
import type { KyouViewPropsBase } from "./kyou-view-props-base"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"

export interface LantanaViewProps extends KyouViewPropsBase {
    height: number | string
    width: number | string
}
