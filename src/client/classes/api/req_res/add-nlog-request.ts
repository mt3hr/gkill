'use strict'

import { Nlog } from '@/classes/datas/nlog'
import { GkillAPIRequest } from '../gkill-api-request'
import type { Kyou } from '@/classes/datas/kyou'

export class AddNlogRequest extends GkillAPIRequest {

    nlog: Nlog

    tx_id: string | null = null

    want_response_kyou: boolean

    added_kyou: Kyou | null = null

    constructor() {
        super()
        this.nlog = new Nlog()
        this.want_response_kyou = false
    }

}


