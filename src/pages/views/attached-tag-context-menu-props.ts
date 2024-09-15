'use strict';

import type { Tag } from "@/classes/datas/tag";
import type { GkillPropsBase } from "./gkill-props-base";

export interface AttachedTagContextMenuProps extends GkillPropsBase {
    tag: Tag;
}