'use strict'

import { GkillAPIResponse } from '../gkill-api-response'
import type { Kyou } from '@/classes/datas/kyou'

export class ReloadRepositoriesResponse extends GkillAPIResponse {

    playing_timeis_kyous: Array<Kyou>

    constructor() {
        super()
        this.playing_timeis_kyous = new Array<Kyou>()
    }

}


