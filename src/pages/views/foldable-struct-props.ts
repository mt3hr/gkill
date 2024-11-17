'use strict'

import type { FoldableStructModel } from "./foldable-struct-model"
import type { GkillPropsBase } from "./gkill-props-base"

export interface FoldableStructProps extends GkillPropsBase {
    struct_obj: FoldableStructModel
    folder_name: string
    is_open: boolean
    is_editable: boolean
    is_show_checkbox: boolean
    is_root: boolean
}
