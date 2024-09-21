'use strict'

import { Mi } from '@/classes/datas/mi'
import { GkillAPIResponse } from '../gkill-api-response'
import { Kyou } from '@/classes/datas/kyou'

export class AddMiResponse extends GkillAPIResponse {

    added_mi: Mi

    added_mi_kyou: Kyou

    constructor() {
        super()
        this.added_mi = new Mi()
        this.added_mi_kyou = new Kyou()
    }

}


