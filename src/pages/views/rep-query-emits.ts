'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"

export interface RepQueryEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'request_clear_rep_query'): void
    (e: 'request_update_checked_reps', checked_reps: Array<string>, is_by_user: boolean): void
    (e: 'request_update_checked_devices', checked_devices: Array<string>, is_by_user: boolean): void
    (e: 'request_update_checked_rep_types', checked_rep_types: Array<string>, is_by_user: boolean): void
    (e: 'inited'): void
}
