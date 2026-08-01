'use strict'

import { MiReKyou } from '@/classes/datas/mi-re-kyou'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetMiReKyouResponse extends GkillAPIResponse {

    mirekyou_histories: Array<MiReKyou>

    constructor() {
        super()
        this.mirekyou_histories = new Array<MiReKyou>()
    }

}
