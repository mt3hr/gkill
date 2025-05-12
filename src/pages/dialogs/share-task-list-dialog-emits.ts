'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { ShareKyouListInfo } from "@/classes/datas/share-kyou-list-info"

export interface ShareKyousListDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'regestered_share_kyou_list_info', share_kyou_list_info: ShareKyouListInfo): void
    (e: 'requested_show_share_kyou_link_dialog', share_kyou_list_info: ShareKyouListInfo): void
}
