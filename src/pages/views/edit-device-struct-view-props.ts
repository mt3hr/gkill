'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import { DeviceStruct } from "@/classes/datas/config/device-struct"

export interface EditDeviceStructViewProps extends GkillPropsBase {
    device_struct: Array<DeviceStruct>
}
