'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { DeviceStructElementData } from "@/classes/datas/config/device-struct-element-data"

export interface AddNewDeviceStructElementViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_add_device_struct_element', device_struct_element: DeviceStructElementData): void
    (e: 'requested_close_dialog'): void
}
