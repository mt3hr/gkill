'use strict'

import { ReKyou } from '@/classes/datas/re-kyou'
import { GkillAPIRequest } from '../gkill-api-request'
import { Kyou } from '@/classes/datas/kyou'

export class UpdateReKyouRequest extends GkillAPIRequest {

    rekyou: ReKyou

    tx_id: string | null = null

    want_response_kyou: boolean

    updated_kyou: Kyou | null | null = null

    constructor() {
        super()
        this.rekyou = new ReKyou()
        this.want_response_kyou = false
    }

}


