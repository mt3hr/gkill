'use strict';

import type { Tag } from "@/classes/datas/tag";
import type { GkillPropsBase } from "./gkill-props-base";
import type { Kyou } from "@/classes/datas/kyou";

export interface AttachedTagContextMenuProps extends GkillPropsBase {
    tag: Tag;
    kyou: Kyou;
    last_added_tag: string;
}
