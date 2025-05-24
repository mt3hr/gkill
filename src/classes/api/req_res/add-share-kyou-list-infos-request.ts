'use strict'

import { ShareKyousInfo } from '@/classes/datas/share-kyous-info'
import { GkillAPIRequest } from '../gkill-api-request'

export class AddShareKyouListInfoRequest extends GkillAPIRequest {

    share_kyou_list_info: ShareKyousInfo

    tx_id: string | null = null

    constructor() {
        super()
        this.share_kyou_list_info = new ShareKyousInfo()
    }

}


