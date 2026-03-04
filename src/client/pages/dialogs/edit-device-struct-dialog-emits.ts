'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { DeviceStructElementData } from "@/classes/datas/config/device-struct-element-data"

export interface EditDeviceStructDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_reload_application_config'): void
    (e: 'requested_apply_device_struct', device_struct_element_data: DeviceStructElementData): void
}
