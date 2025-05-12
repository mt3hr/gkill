'use strict'

import type { ShareKyouListInfo } from "@/classes/datas/share-kyou-list-info"

export interface ConfirmDeleteShareKyousListViewEmits {
    (e: 'requested_delete_share_kyou_link_info', share_kyou_link_info: ShareKyouListInfo): void
    (e: 'requested_close_dialog'): void
}
