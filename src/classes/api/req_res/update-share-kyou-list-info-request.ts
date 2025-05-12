'use strict'

import { ShareKyouListInfo } from '@/classes/datas/share-kyou-list-info'
import { GkillAPIRequest } from '../gkill-api-request'

export class UpdateShareKyouListInfoRequest extends GkillAPIRequest {

    share_kyou_list_info: ShareKyouListInfo

    constructor() {
        super()
        this.share_kyou_list_info = new ShareKyouListInfo()
    }

}


