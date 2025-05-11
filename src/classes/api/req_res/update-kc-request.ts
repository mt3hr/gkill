'use strict'

import { KC } from '@/classes/datas/kc'
import { GkillAPIRequest } from '../gkill-api-request'

export class UpdateKCRequest extends GkillAPIRequest {

    kc: KC

    constructor() {
        super()
        this.kc = new KC()
    }

}


