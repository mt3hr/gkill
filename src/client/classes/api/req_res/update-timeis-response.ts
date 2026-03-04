'use strict'

import { TimeIs } from '@/classes/datas/time-is'
import { GkillAPIResponse } from '../gkill-api-response'
import { Kyou } from '@/classes/datas/kyou'

export class UpdateTimeisResponse extends GkillAPIResponse {

    updated_timeis: TimeIs

    updated_kyou: Kyou | null

    constructor() {
        super()
        this.updated_timeis = new TimeIs()
        this.updated_kyou = null
    }

}


