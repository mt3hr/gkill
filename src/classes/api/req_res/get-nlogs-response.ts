'use strict'

import { Nlog } from '@/classes/datas/nlog'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetNlogsResponse extends GkillAPIResponse {

    nlogs: Array<Nlog>

    constructor() {
        super()
        this.nlogs = new Array<Nlog>()
    }

}


