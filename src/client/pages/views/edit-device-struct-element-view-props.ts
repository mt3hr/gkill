'use strict'

import type { DeviceStructElementData } from "@/classes/datas/config/device-struct-element-data"
import type { GkillPropsBase } from "./gkill-props-base"

export interface EditDeviceStructElementViewProps extends GkillPropsBase {
    struct_obj: DeviceStructElementData
}
