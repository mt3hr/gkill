'use strict';

import type { Kyou } from "@/classes/datas/kyou";
import type { GkillPropsBase } from "./gkill-props-base";

export interface DnoteNlogsListViewProps extends GkillPropsBase {
    nlog_kyous: Array<Kyou>;
}
