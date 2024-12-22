'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { KFTLTemplateElementData } from "@/classes/datas/kftl-template-element-data"

export interface KFTLTemplateViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'clicked_template_element_leaf', template_leaf: KFTLTemplateElementData): void
    (e: 'requested_close_dialog'): void
}
