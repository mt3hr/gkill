'use strict'

import { Lantana } from '@/classes/datas/lantana'
import { GkillAPIRequest } from '../gkill-api-request'
import { Kyou } from '@/classes/datas/kyou'

export class UpdateLantanaRequest extends GkillAPIRequest {

    lantana: Lantana

    tx_id: string | null = null

    want_response_kyou: boolean

    updated_kyou: Kyou | null | null = null

    constructor() {
        super()
        this.lantana = new Lantana()
        this.want_response_kyou = false
    }

}


