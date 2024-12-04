'use strict'

import type { Kmemo } from "@/classes/datas/kmemo"
import type { KyouViewPropsBase } from "./kyou-view-props-base"
import type { InfoIdentifier } from "@/classes/datas/info-identifier"

export interface KmemoViewProps extends KyouViewPropsBase {
    kmemo: Kmemo
    height: number | string
    width: number | string
}
