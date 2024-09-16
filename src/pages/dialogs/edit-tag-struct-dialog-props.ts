'use strict';

import type { TagStruct } from "@/classes/datas/config/tag-struct";
import type { GkillPropsBase } from "../views/gkill-props-base";

export interface EditTagStructDialogProps extends GkillPropsBase {
    tag_struct: TagStruct
    tag_struct_root: TagStruct
    app_content_height: Number;
    app_content_width: Number;
}
