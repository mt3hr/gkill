'use strict'

import { ShareKyouListInfo } from '@/classes/datas/share-kyou-list-info'
import { GkillAPIResponse } from '../gkill-api-response'

export class UpdateShareKyouListInfoResponse extends GkillAPIResponse {

    share_kyou_list_info: ShareKyouListInfo

    constructor() {
        super()
        this.share_kyou_list_info = new ShareKyouListInfo()
    }

}


