'use strict';

import type { Kyou } from "@/classes/datas/kyou";
import type { GkillPropsBase } from "./gkill-props-base";

export interface DnoteNlogViewProps extends GkillPropsBase {
    nlog_kyou: Kyou;
    highlight_targets: Array<Kyou>;
    last_added_tag: string;
}
