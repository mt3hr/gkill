'use strict'

import { Kmemo } from '@/classes/datas/kmemo'
import { GkillAPIRequest } from '../gkill-api-request'

export class AddKmemoRequest extends GkillAPIRequest {

    kmemo: Kmemo

    tx_id: string | null = null

    constructor() {
        super()
        this.kmemo = new Kmemo()
    }

}


