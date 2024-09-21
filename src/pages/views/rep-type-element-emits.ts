'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { CheckState } from "./check-state"

export interface RepTypeElementEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'clicked_is_check_when_inited', items: Array<string>, is_by_user: boolean): void
    (e: 'requested_update_is_check_when_inited_state', items: Array<string>, check_state: CheckState): void
}
