'use strict'

import { Text } from '@/classes/datas/text'
import { GkillAPIRequest } from '../gkill-api-request'

export class AddTextRequest extends GkillAPIRequest {

    text: Text

    tx_id: string | null = null

    constructor() {
        super()
        this.text = new Text()
    }

}


