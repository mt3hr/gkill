'use strict';

import type { GkillPropsBase } from "./gkill-props-base";
import { TagStruct } from "@/classes/datas/config/tag-struct";

export interface EditTagStructViewProps extends GkillPropsBase {
    tag_struct_root: Array<TagStruct>
    tag_struct: Array<TagStruct>
}
