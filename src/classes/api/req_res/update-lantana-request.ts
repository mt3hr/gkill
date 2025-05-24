'use strict'

import { Lantana } from '@/classes/datas/lantana'
import { GkillAPIRequest } from '../gkill-api-request'

export class UpdateLantanaRequest extends GkillAPIRequest {

    lantana: Lantana

    tx_id: string | null = null

    constructor() {
        super()
        this.lantana = new Lantana()
    }

}


