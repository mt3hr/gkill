'use strict'

import { TimeIs } from '@/classes/datas/time-is'
import { GkillAPIResponse } from '../gkill-api-response'
import type { Kyou } from '@/classes/datas/kyou'

export class GetPlaingTimeisResponse extends GkillAPIResponse {

    plaing_timeis_kyous: Array<Kyou>

    constructor() {
        super()
        this.plaing_timeis_kyous = new Array<Kyou>()
    }

}


