'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { DeviceStruct } from "@/classes/datas/config/device-struct"

export interface EditDeviceStructElementViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_update_device_struct', device_struct: DeviceStruct): void
    (e: 'requested_close_dialog'): void
}
