'use strict'

import { MiReKyou } from '@/classes/datas/mi-re-kyou'
import { GkillAPIResponse } from '../gkill-api-response'
import { Kyou } from '@/classes/datas/kyou'

export class UpdateMiReKyouResponse extends GkillAPIResponse {

    updated_mirekyou: MiReKyou

    updated_kyou: Kyou | null

    constructor() {
        super()
        this.updated_mirekyou = new MiReKyou()
        this.updated_kyou = null
    }

}
