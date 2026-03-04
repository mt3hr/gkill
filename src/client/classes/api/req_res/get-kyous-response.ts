'use strict'

import { Kyou } from '@/classes/datas/kyou'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetKyousResponse extends GkillAPIResponse {

    kyous: Array<Kyou>

    constructor() {
        super()
        this.kyous = new Array<Kyou>()
    }

}


