'use strict'

import { Kyou } from '@/classes/datas/kyou'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetKyouResponse extends GkillAPIResponse {

    kyou_histories: Array<Kyou>

    constructor() {
        super()
        this.kyou_histories = new Array<Kyou>()
    }

}


