'use strict';

import type { Kyou } from "@/classes/datas/kyou";
import type { GkillPropsBase } from "./gkill-props-base";

export interface DnotePeoplesListViewProps extends GkillPropsBase {
    timeis_kyous: Array<Kyou>;
}
