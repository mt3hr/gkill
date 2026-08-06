'use strict'

import { MiReKyou } from '@/classes/datas/mi-re-kyou'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetMiReKyousByTargetIDResponse extends GkillAPIResponse {

    mirekyous: Array<MiReKyou>

    constructor() {
        super()
        this.mirekyous = new Array<MiReKyou>()
    }

}
