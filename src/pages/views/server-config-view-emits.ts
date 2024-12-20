'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { ServerConfig } from "@/classes/datas/config/server-config"

export interface ServerConfigViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_reload_server_config'): void
    (e: 'requested_close_dialog'): void
}
