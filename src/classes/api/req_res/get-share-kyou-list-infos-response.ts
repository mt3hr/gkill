'use strict'

import { ShareKyousInfo } from '@/classes/datas/share-kyous-info'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetShareKyouListInfosResponse extends GkillAPIResponse {

    share_kyou_list_infos: Array<ShareKyousInfo>

    constructor() {
        super()
        this.share_kyou_list_infos = new Array<ShareKyousInfo>()
    }

}


