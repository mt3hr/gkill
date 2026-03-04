'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { ApplicationConfig } from "@/classes/datas/config/application-config"
import type { KFTLTemplateStructElementData } from "@/classes/datas/config/kftl-template-struct-element-data"

export interface EditKFTLTemplateStructViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_reload_application_config', application_config: ApplicationConfig): void
    (e: 'requested_apply_kftl_template_struct', kftl_template_struct_element_data: KFTLTemplateStructElementData): void
    (e: 'requested_close_dialog'): void
}
