'use strict';

import type { Tag } from "@/classes/datas/tag";
import type { KyouViewPropsBase } from "../views/kyou-view-props-base";

export interface TagHistoriesDialogProps extends KyouViewPropsBase {
    tag: Tag;
    is_show: boolean;
}
