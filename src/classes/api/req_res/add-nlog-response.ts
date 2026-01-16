'use strict'

import { Nlog } from '@/classes/datas/nlog'
import { GkillAPIResponse } from '../gkill-api-response'
import { Kyou } from '@/classes/datas/kyou'

export class AddNlogResponse extends GkillAPIResponse {

    added_nlog: Nlog

    added_kyou: Kyou | null

    constructor() {
        super()
        this.added_nlog = new Nlog()
        this.added_kyou = null
    }

}


