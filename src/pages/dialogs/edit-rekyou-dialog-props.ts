'use strict';

import type { ReKyou } from "@/classes/datas/re-kyou";
import type { KyouViewPropsBase } from "../views/kyou-view-props-base";

export interface EditRekyouDialogProps extends KyouViewPropsBase {
    rekyou: ReKyou;
    is_show: boolean;
}
