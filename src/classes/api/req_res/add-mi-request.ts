'use strict'

import { Mi } from '@/classes/datas/mi'
import { GkillAPIRequest } from '../gkill-api-request'
import type { Kyou } from '@/classes/datas/kyou'

export class AddMiRequest extends GkillAPIRequest {

    mi: Mi

    tx_id: string | null = null

    want_response_kyou: boolean

    added_kyou: Kyou | null = null

    constructor() {
        super()
        this.mi = new Mi()
        this.want_response_kyou = false
    }

}


