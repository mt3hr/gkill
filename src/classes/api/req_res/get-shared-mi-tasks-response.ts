'use strict'

import { Kyou } from '@/classes/datas/kyou'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetSharedMiTasksResponse extends GkillAPIResponse {

    mi_kyous: Array<Kyou>

    constructor() {
        super()
        this.mi_kyous = new Array<Kyou>()
    }

}


