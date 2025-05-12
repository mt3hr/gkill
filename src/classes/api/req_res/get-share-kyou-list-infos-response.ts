'use strict'

import { ShareKyouListInfo } from '@/classes/datas/share-kyou-list-info'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetShareKyouListInfosResponse extends GkillAPIResponse {

    share_kyou_list_infos: Array<ShareKyouListInfo>

    constructor() {
        super()
        this.share_kyou_list_infos = new Array<ShareKyouListInfo>()
    }

}


