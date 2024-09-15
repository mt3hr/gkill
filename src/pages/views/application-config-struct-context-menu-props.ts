'use strict';

import type { GkillPropsBase } from "./gkill-props-base";

export interface ApplicationConfigStructContextMenuProps extends GkillPropsBase {
    struct_obj: Object;
    folder_name: string;
    is_open: boolean;
}
