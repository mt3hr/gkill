'use strict'

import type { Lantana } from "@/classes/datas/lantana"
import type { KyouViewPropsBase } from "./kyou-view-props-base"

export interface EditLantanaViewProps extends KyouViewPropsBase {
    lantana: Lantana
}
