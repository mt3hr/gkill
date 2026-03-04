'use strict'

import type { IDFKyou } from '@/classes/datas/idf-kyou'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetIDFKyouResponse extends GkillAPIResponse {

    idf_kyou_histories: Array<IDFKyou>

    constructor() {
        super()
        this.idf_kyou_histories= new Array<IDFKyou>()
    }

}


