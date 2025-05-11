'use strict'

import { KC } from '@/classes/datas/kc'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetKCResponse extends GkillAPIResponse {

    kc_histories: Array<KC>

    constructor() {
        super()
        this.kc_histories = new Array<KC>()
    }

}


