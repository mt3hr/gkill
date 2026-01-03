'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { TagStructElementData } from "@/classes/datas/config/tag-struct-element-data"

export interface EditTagStructDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_apply_tag_struct', tag_struct_element_data: TagStructElementData): void
    (e: 'requested_reload_application_config'): void
}
