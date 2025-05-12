'use strict'

import type { ShareKyouListInfo } from "@/classes/datas/share-kyou-list-info"

export interface ManageShareKyousListViewEmits {
    (e: 'requested_show_share_kyou_link_dialog', share_kyou_list_info: ShareKyouListInfo): void
    (e: 'requested_show_confirm_delete_share_kyou_list_dialog', share_kyou_list_info: ShareKyouListInfo): void
}
