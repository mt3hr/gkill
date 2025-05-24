'use strict'

import { KC } from '@/classes/datas/kc'
import { GkillAPIRequest } from '../gkill-api-request'

export class AddKCRequest extends GkillAPIRequest {

    kc: KC

    tx_id: string | null = null

    constructor() {
        super()
        this.kc = new KC()
    }

}


