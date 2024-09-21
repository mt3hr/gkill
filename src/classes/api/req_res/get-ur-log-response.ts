'use strict'

import { URLog } from '@/classes/datas/ur-log'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetURLogResponse extends GkillAPIResponse {

    urlog_histories: Array<URLog>

    constructor() {
        super()
        this.urlog_histories = new Array<URLog>()
    }

}


