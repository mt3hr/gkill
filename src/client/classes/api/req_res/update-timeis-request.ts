'use strict'

import { TimeIs } from '@/classes/datas/time-is'
import { GkillAPIRequest } from '../gkill-api-request'
import { Kyou } from '@/classes/datas/kyou'

export class UpdateTimeisRequest extends GkillAPIRequest {

    timeis: TimeIs

    tx_id: string | null = null

    want_response_kyou: boolean

    updated_kyou: Kyou | null | null = null

    constructor() {
        super()
        this.timeis = new TimeIs()
        this.want_response_kyou = false
    }

}


