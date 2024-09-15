'use strict';

import type { LineLabelData } from "@/classes/kftl/line-label-data";
import type { GkillPropsBase } from "./gkill-props-base";

export interface KFTLLineLabelProps extends GkillPropsBase {
    style: any;
    line_label_data: LineLabelData;
}
