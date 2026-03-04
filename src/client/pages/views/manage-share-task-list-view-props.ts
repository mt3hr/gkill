'use strict'

import type { ShareKyousInfo } from "@/classes/datas/share-kyous-info"
import type { GkillPropsBase } from "./gkill-props-base"

export interface ManageShareKyousListViewProps extends GkillPropsBase {
    share_kyou_list_infos: Array<ShareKyousInfo>
}
