'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { MiBoardStructElementData } from "@/classes/datas/config/mi-board-struct-element-data"

export interface EditMiBoardStructDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_apply_mi_board_struct', mi_board_struct_element_data: MiBoardStructElementData): void
    (e: 'requested_reload_application_config'): void
}
