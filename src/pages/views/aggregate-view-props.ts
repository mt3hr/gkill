'use strict';

import type { Kyou } from "@/classes/datas/kyou";
import type { GkillPropsBase } from "./gkill-props-base";

export interface AggregateViewProps extends GkillPropsBase {
    checked_kyous: Array<Kyou>;
}
