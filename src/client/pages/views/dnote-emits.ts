'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { ApplicationConfig } from "@/classes/datas/config/application-config"
import type { KyouViewEmits } from "./kyou-view-emits"

export interface DnoteEmits extends KyouViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_close_dialog'): void
    (e: 'requested_apply_dnote', dnote_data: any): void
    (e: 'requested_reload_application_config', application_config: ApplicationConfig): void
}
