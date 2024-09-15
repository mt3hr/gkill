'use strict';

import type { Kmemo } from "@/classes/datas/kmemo";
import type { KyouViewPropsBase } from "./kyou-view-props-base";

export interface EditKmemoViewProps extends KyouViewPropsBase {
    kmemo: Kmemo;
}