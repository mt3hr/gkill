'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { KFTLTemplateElementData } from "@/classes/datas/kftl-template-element-data"

export interface EditKFTLTemplateStructElementDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'requested_update_kftl_template_struct', kftl_template_struct: KFTLTemplateElementData): void
    (e: 'received_errors', errors: Array<GkillError>): void
}
