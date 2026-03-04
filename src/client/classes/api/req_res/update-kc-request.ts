'use strict'

import { KC } from '@/classes/datas/kc'
import { GkillAPIRequest } from '../gkill-api-request'
import { Kyou } from '@/classes/datas/kyou'

export class UpdateKCRequest extends GkillAPIRequest {

    kc: KC

    tx_id: string | null = null

    want_response_kyou: boolean

    updated_kyou: Kyou | null | null = null

    constructor() {
        super()
        this.kc = new KC()
        this.want_response_kyou = false
    }

}


