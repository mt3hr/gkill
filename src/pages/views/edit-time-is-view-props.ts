'use strict';

import type { TimeIs } from "@/classes/datas/time-is";
import type { KyouViewPropsBase } from "./kyou-view-props-base";

export interface EditTimeIsViewProps extends KyouViewPropsBase {
    timeis: TimeIs;
}
