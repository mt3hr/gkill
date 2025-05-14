'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { ShareKyousInfo } from "@/classes/datas/share-kyous-info"

export interface ShareKyousLinkViewEmits {
    (e: 'updated_share_kyou_list_info', share_kyou_list_info: ShareKyousInfo): void
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_close_dialog'): void
}
