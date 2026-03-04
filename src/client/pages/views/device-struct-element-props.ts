'use strict'

import type { GkillPropsBase } from "./gkill-props-base"

export interface DeviceStructElementProps extends GkillPropsBase {
    struct_obj: Object
    folder_name: string
    is_open: boolean
}
