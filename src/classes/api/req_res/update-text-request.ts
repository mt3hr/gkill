'use strict'

import { Text } from '@/classes/datas/text'
import { GkillAPIRequest } from '../gkill-api-request'

export class UpdateTextRequest extends GkillAPIRequest {

    text: Text

    constructor() {
        super()
        this.text = new Text()
    }

}


