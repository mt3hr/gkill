'use strict'

import { IDFKyou } from '@/classes/datas/idf-kyou'
import { GkillAPIResponse } from '../gkill-api-response'
import { Kyou } from '@/classes/datas/kyou'

export class UpdateIDFKyouResponse extends GkillAPIResponse {

    updated_idf_kyou: IDFKyou

    updated_idf_kyou_kyou: Kyou

    constructor() {
        super()
        this.updated_idf_kyou = new IDFKyou()
        this.updated_idf_kyou_kyou = new Kyou()
    }

}


