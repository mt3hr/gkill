'use strict'

import { TimeIs } from '@/classes/datas/time-is'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetTimeisResponse extends GkillAPIResponse {

    timeis_histories: Array<TimeIs>

    constructor() {
        super()
        this.timeis_histories = new Array<TimeIs>()
    }

}


