'use strict'

import type { ShareKyousInfo } from "@/classes/datas/share-kyous-info"

export interface ManageShareKyousListViewEmits {
    (e: 'requested_show_share_kyou_link_dialog', share_kyou_list_info: ShareKyousInfo): void
    (e: 'requested_show_confirm_delete_share_kyou_list_dialog', share_kyou_list_info: ShareKyousInfo): void
}
