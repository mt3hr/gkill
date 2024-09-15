'use strict';

import type { GkillPropsBase } from "./gkill-props-base";
import type { SidebarProps } from "./sidebar-props";

export interface FoldableStructProps extends SidebarProps, GkillPropsBase {
    struct_obj: Object;
    folder_name: string;
    is_open: boolean;
}