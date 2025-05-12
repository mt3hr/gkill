'use strict'

import type { ShareKyouListInfo } from "@/classes/datas/share-kyou-list-info"
import type { GkillPropsBase } from "./gkill-props-base"

export interface ConfirmDeleteShareKyousListViewProps extends GkillPropsBase {
    share_kyou_list_info: ShareKyouListInfo
}
