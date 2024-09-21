'use strict'

import { Kmemo } from '@/classes/datas/kmemo'
import { GkillAPIResponse } from '../gkill-api-response'
import { Kyou } from '@/classes/datas/kyou'

export class UpdateKmemoResponse extends GkillAPIResponse {

    updated_kmemo: Kmemo

    updated_kmemo_kyou: Kyou

    constructor() {
        super()
        this.updated_kmemo = new Kmemo()
        this.updated_kmemo_kyou = new Kyou()
    }

}


