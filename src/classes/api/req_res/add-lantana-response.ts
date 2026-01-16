'use strict'

import { Lantana } from '@/classes/datas/lantana'
import { GkillAPIResponse } from '../gkill-api-response'
import { Kyou } from '@/classes/datas/kyou'

export class AddLantanaResponse extends GkillAPIResponse {

    added_lantana: Lantana

    added_kyou: Kyou | null

    constructor() {
        super()
        this.added_lantana = new Lantana()
        this.added_kyou = null
    }

}


