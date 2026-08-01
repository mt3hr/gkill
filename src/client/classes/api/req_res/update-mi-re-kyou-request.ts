'use strict'

import { MiReKyou } from '@/classes/datas/mi-re-kyou'
import { GkillAPIRequest } from '../gkill-api-request'
import { Kyou } from '@/classes/datas/kyou'

export class UpdateMiReKyouRequest extends GkillAPIRequest {

    mirekyou: MiReKyou

    tx_id: string | null = null

    want_response_kyou: boolean

    updated_kyou: Kyou | null = null

    constructor() {
        super()
        this.mirekyou = new MiReKyou()
        this.want_response_kyou = false
    }

}
