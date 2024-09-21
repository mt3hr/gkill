'use strict'

import { Mi } from '@/classes/datas/mi'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetMisResponse extends GkillAPIResponse {

    mis: Array<Mi>

    constructor() {
        super()
        this.mis = new Array<Mi>()
    }

}


