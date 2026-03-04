'use strict'

import type { Kyou } from '@/classes/datas/kyou'
import { GkillAPIResponse } from '../gkill-api-response'
import { IDFKyou } from '@/classes/datas/idf-kyou'

export class AddKyouInfoResponse extends GkillAPIResponse {

    added_idf_kyou: IDFKyou

    added_kyou: Kyou | null

    constructor() {
        super()
        this.added_idf_kyou = new IDFKyou()
        this.added_kyou = null
    }

}


