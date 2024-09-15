'use strict';

import type { KyouViewPropsBase } from "../views/kyou-view-props-base";
import { Text } from "@/classes/datas/text";

export interface TextHistoriesDialogProps extends KyouViewPropsBase {
    text: Text;
    is_show: boolean;
}
