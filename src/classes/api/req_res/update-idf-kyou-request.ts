'use strict'

import { GkillAPIRequest } from '../gkill-api-request'
import { IDFKyou } from '@/classes/datas/idf-kyou'

export class UpdateIDFKyouRequest extends GkillAPIRequest {

    idf_kyou: IDFKyou

    constructor() {
        super()
        this.idf_kyou = new IDFKyou()
    }

}


