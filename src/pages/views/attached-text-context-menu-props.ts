'use strict';

import type { GkillPropsBase } from "./gkill-props-base";
import { Text } from "@/classes/datas/text";

export interface AttachedTextContextMenuProps extends GkillPropsBase {
    text: Text;
}
