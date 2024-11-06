'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { CheckState } from "./check-state"

export interface FoldableStructEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'clicked_items', items: Array<string>, check_state: CheckState, is_by_user: boolean): void
    (e: 'requested_update_check_state', items: Array<string>, check_state: CheckState): void
}
