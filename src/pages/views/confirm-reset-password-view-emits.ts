'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { Account } from "@/classes/datas/config/account"
import type { ServerConfig } from "@/classes/datas/config/server-config"

export interface ConfirmResetPasswordViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'reseted_password', user_id: string, password_reset_path_without_host: string): void
    (e: 'requested_show_show_password_reset_dialog', account: Account): void
    (e: 'requested_reload_server_config'): void
    (e: 'requested_close_dialog'): void
}
