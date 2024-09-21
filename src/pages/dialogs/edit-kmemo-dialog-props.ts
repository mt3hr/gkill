'use strict'

import type { Kmemo } from "@/classes/datas/kmemo"
import type { KyouViewPropsBase } from "../views/kyou-view-props-base"

export interface EditKmemoDialogProps extends KyouViewPropsBase {
    kmemo: Kmemo
   
}
