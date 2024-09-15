'use strict';

import type { Tag } from "@/classes/datas/tag";
import type { KyouViewPropsBase } from "./kyou-view-props-base";

export interface EditTagViewProps extends KyouViewPropsBase {
    tag: Tag;
}
