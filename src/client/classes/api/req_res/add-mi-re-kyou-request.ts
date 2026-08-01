'use strict'

import { MiReKyou } from '@/classes/datas/mi-re-kyou'
import { GkillAPIRequest } from '../gkill-api-request'
import type { Kyou } from '@/classes/datas/kyou'

export class AddMiReKyouRequest extends GkillAPIRequest {

    mirekyou: MiReKyou

    tx_id: string | null = null

    want_response_kyou: boolean

    added_kyou: Kyou | null = null

    constructor() {
        super()
        this.mirekyou = new MiReKyou()
        this.want_response_kyou = false
    }

}
