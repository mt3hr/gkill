'use strict'

import { TimeIs } from '@/classes/datas/time-is'
import { GkillAPIRequest } from '../gkill-api-request'

export class AddTimeisRequest extends GkillAPIRequest {

    timeis: TimeIs

    tx_id: string | null = null

    constructor() {
        super()
        this.timeis = new TimeIs()
    }

}


