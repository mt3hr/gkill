'use strict'

import { Kmemo } from '@/classes/datas/kmemo'
import { GkillAPIRequest } from '../gkill-api-request'
import { Kyou } from '@/classes/datas/kyou'

export class UpdateKmemoRequest extends GkillAPIRequest {

    kmemo: Kmemo

    tx_id: string | null = null

    want_response_kyou: boolean

    updated_kyou: Kyou | null | null = null

    constructor() {
        super()
        this.kmemo = new Kmemo()
        this.want_response_kyou = false
    }

}


