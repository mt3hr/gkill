'use strict'

import { ReKyou } from '@/classes/datas/re-kyou'
import { GkillAPIRequest } from '../gkill-api-request'

export class UpdateReKyouRequest extends GkillAPIRequest {

    rekyou: ReKyou

    constructor() {
        super()
        this.rekyou = new ReKyou()
    }

}


