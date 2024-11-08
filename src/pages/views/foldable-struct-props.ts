'use strict'

import type { FoldableStructModel } from "./foldable-struct-model"
import type { GkillPropsBase } from "./gkill-props-base"
import type { SidebarProps } from "./sidebar-props"

export interface FoldableStructProps extends SidebarProps, GkillPropsBase {
    struct_obj: FoldableStructModel
    folder_name: string
    is_open: boolean
}
