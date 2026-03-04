'use strict'

import { ReKyou } from '@/classes/datas/re-kyou'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetReKyouResponse extends GkillAPIResponse {

    rekyou_histories: Array<ReKyou>

    constructor() {
        super()
        this.rekyou_histories = new Array<ReKyou>()
    }

}


