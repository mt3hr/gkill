'use strict'

import { ReKyou } from '@/classes/datas/re-kyou'
import { GkillAPIResponse } from '../gkill-api-response'
import { Kyou } from '@/classes/datas/kyou'

export class AddReKyouResponse extends GkillAPIResponse {

    added_rekyou: ReKyou

    added_rekyou_kyou: Kyou

    constructor() {
        super()
        this.added_rekyou = new ReKyou()
        this.added_rekyou_kyou = new Kyou()
    }

}


