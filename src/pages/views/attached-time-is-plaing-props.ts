'use strict';

import type { TimeIs } from "@/classes/datas/time-is";
import type { GkillPropsBase } from "./gkill-props-base";

export interface AttachedTimeIsPlaingProps extends GkillPropsBase {
    timeis: TimeIs;
}
