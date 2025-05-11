'use strict'

import { KC } from '@/classes/datas/kc'
import { GkillAPIResponse } from '../gkill-api-response'
import { Kyou } from '@/classes/datas/kyou'

export class AddKCResponse extends GkillAPIResponse {

    added_kc: KC

    added_kc_kyou: Kyou

    constructor() {
        super()
        this.added_kc = new KC()
        this.added_kc_kyou = new Kyou()
    }

}


