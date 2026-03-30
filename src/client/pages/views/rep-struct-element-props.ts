'use strict'

import type { GkillPropsBase } from "./gkill-props-base"

export interface RepStructElementProps extends GkillPropsBase {
    struct_obj: object
    folder_name: string
    is_open: boolean
}
