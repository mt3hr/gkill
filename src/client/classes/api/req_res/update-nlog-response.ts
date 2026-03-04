'use strict'

import { Nlog } from '@/classes/datas/nlog'
import { GkillAPIResponse } from '../gkill-api-response'
import { Kyou } from '@/classes/datas/kyou'

export class UpdateNlogResponse extends GkillAPIResponse {

    updated_nlog: Nlog

    updated_kyou: Kyou | null

    constructor() {
        super()
        this.updated_nlog = new Nlog()
        this.updated_kyou = null
    }

}


