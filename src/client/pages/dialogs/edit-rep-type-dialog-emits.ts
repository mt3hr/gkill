'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { ApplicationConfig } from "@/classes/datas/config/application-config"
import type { RepTypeStructElementData } from "@/classes/datas/config/rep-type-struct-element-data"

export interface EditRepTypeDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_reload_application_config', application_config: ApplicationConfig): void
    (e: 'requested_apply_rep_type_struct', rep_type_struct_element_data: RepTypeStructElementData): void
}
