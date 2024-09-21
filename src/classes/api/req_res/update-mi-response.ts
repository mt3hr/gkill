'use strict'

import { Mi } from '@/classes/datas/mi'
import { GkillAPIResponse } from '../gkill-api-response'
import { Kyou } from '@/classes/datas/kyou'

export class UpdateMiResponse extends GkillAPIResponse {

    updated_mi: Mi

    updated_mi_kyou: Kyou

    constructor() {
        super()
        this.updated_mi = new Mi()
        this.updated_mi_kyou = new Kyou()
    }

}


