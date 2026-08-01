'use strict'

import { MiReKyou } from '@/classes/datas/mi-re-kyou'
import { GkillAPIResponse } from '../gkill-api-response'
import { Kyou } from '@/classes/datas/kyou'

export class AddMiReKyouResponse extends GkillAPIResponse {

    added_mirekyou: MiReKyou

    added_kyou: Kyou | null

    constructor() {
        super()
        this.added_mirekyou = new MiReKyou()
        this.added_kyou = null
    }

}
