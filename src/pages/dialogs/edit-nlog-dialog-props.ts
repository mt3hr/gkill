'use strict';

import type { Nlog } from "@/classes/datas/nlog";
import type { KyouViewPropsBase } from "../views/kyou-view-props-base";

export interface EditNlogDialogProps extends KyouViewPropsBase {
    nlog: Nlog;
    is_show: boolean;
}
