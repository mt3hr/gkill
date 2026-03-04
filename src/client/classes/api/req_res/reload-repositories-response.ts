'use strict'

import { GkillAPIResponse } from '../gkill-api-response'
import type { Kyou } from '@/classes/datas/kyou'

export class ReloadRepositoriesResponse extends GkillAPIResponse {

    plaing_timeis_kyous: Array<Kyou>

    constructor() {
        super()
        this.plaing_timeis_kyous = new Array<Kyou>()
    }

}


