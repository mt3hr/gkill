'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { Account } from "@/classes/datas/config/account"

export interface CreateAccountViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'added_account', account: Account): void
    (e: 'requested_reload_server_config'): void
    (e: 'requested_close_dialog'): void
    (e: 'created_account', account: Account): void
}
