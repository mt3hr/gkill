'use strict'

import { Text } from '@/classes/datas/text'
import { GkillAPIResponse } from '../gkill-api-response'

export class UpdateTextResponse extends GkillAPIResponse {

    updated_text: Text

    constructor() {
        super()
        this.updated_text = new Text()
    }

}


