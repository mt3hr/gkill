'use strict'

import type { Kyou } from '@/classes/datas/kyou'
import { GkillAPIRequest } from '../gkill-api-request'
import { IDFKyou } from '@/classes/datas/idf-kyou'

export class UpdateIDFKyouRequest extends GkillAPIRequest {

    idf_kyou: IDFKyou

    tx_id: string | null = null

    want_response_kyou: boolean

    updated_kyou: Kyou | null | null = null

    constructor() {
        super()
        this.idf_kyou = new IDFKyou()
        this.want_response_kyou = true
    }

}


