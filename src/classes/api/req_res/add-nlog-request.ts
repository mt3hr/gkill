'use strict'

import { Nlog } from '@/classes/datas/nlog'
import { GkillAPIRequest } from '../gkill-api-request'

export class AddNlogRequest extends GkillAPIRequest {

    nlog: Nlog

    tx_id: string | null = null

    constructor() {
        super()
        this.nlog = new Nlog()
    }

}


