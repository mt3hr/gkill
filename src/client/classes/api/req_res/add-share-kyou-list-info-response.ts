'use strict'

import { ShareKyousInfo } from '@/classes/datas/share-kyous-info'
import { GkillAPIResponse } from '../gkill-api-response'

export class AddShareKyouListInfoResponse extends GkillAPIResponse {

    share_kyou_list_info: ShareKyousInfo

    constructor() {
        super()
        this.share_kyou_list_info = new ShareKyousInfo()
    }

}


