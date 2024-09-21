'use strict'

import { Text } from '@/classes/datas/text'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetTextsByTargetIDResponse extends GkillAPIResponse {

    texts: Array<Text>

    constructor() {
        super()
        this.texts = new Array<Text>()
    }

}


