'use strict'

import { KC } from '@/classes/datas/kc'
import { GkillAPIResponse } from '../gkill-api-response'
import { Kyou } from '@/classes/datas/kyou'

export class UpdateKCResponse extends GkillAPIResponse {

    updated_kc: KC

    updated_kyou: Kyou | null

    constructor() {
        super()
        this.updated_kc = new KC()
        this.updated_kyou = null
    }

}


