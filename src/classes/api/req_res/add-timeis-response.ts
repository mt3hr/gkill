'use strict'

import { TimeIs } from '@/classes/datas/time-is'
import { GkillAPIResponse } from '../gkill-api-response'
import { Kyou } from '@/classes/datas/kyou'

export class AddTimeisResponse extends GkillAPIResponse {

    added_timeis: TimeIs

    added_timeis_kyou: Kyou

    constructor() {
        super()
        this.added_timeis = new TimeIs()
        this.added_timeis_kyou = new Kyou()
    }

}


