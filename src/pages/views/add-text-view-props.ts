'use strict';

import type { KyouViewPropsBase } from "./kyou-view-props-base";
import { Text } from "@/classes/datas/text";

export interface AddTextViewProps extends KyouViewPropsBase {
    text: Text;
}
