'use strict'

import type { GkillPropsBase } from "./gkill-props-base"
import type { DeviceStruct } from "@/classes/datas/config/device-struct"

export interface ConfirmDeleteDeviceStructViewProps extends GkillPropsBase {
    device_struct: DeviceStruct
}
