'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { KFTLTemplateStructElementData } from "@/classes/datas/config/kftl-template-struct-element-data"

export interface AddNewKFTLTemplateStructElementDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_add_kftl_template_struct_element', kftl_template_struct_element: KFTLTemplateStructElementData): void
}
