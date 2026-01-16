'use strict'

import { KC } from '@/classes/datas/kc'
import { GkillAPIResponse } from '../gkill-api-response'
import { Kyou } from '@/classes/datas/kyou'

export class AddKCResponse extends GkillAPIResponse {

    added_kc: KC

    added_kyou: Kyou | null

    constructor() {
        super()
        this.added_kc = new KC()
        this.added_kyou = null
    }

}


