'use strict';

import type { Tag } from "@/classes/datas/tag";
import type { KyouViewPropsBase } from "./kyou-view-props-base";

export interface ConfirmDeleteTagViewProps extends KyouViewPropsBase {
    tag: Tag;
}
