'use strict'

import type { ShareKyousInfo } from "@/classes/datas/share-kyous-info"

export interface ConfirmDeleteShareKyousListViewEmits {
    (e: 'requested_delete_share_kyou_link_info', share_kyou_link_info: ShareKyousInfo): void
    (e: 'requested_close_dialog'): void
}
