'use strict';

import type { IDFKyou } from "@/classes/datas/idf-kyou";
import type { KyouViewPropsBase } from "../views/kyou-view-props-base";

export interface EditIDFKyouDialogProps extends KyouViewPropsBase {
    idf_kyou: IDFKyou;
    is_show: boolean;
}
