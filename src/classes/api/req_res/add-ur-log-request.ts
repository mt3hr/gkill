'use strict'

import { URLog } from '@/classes/datas/ur-log'
import { GkillAPIRequest } from '../gkill-api-request'
import type { Kyou } from '@/classes/datas/kyou'

export class AddURLogRequest extends GkillAPIRequest {

    urlog: URLog

    tx_id: string | null = null

    want_response_kyou: boolean

    added_kyou: Kyou | null = null

    constructor() {
        super()
        this.urlog = new URLog()
        this.want_response_kyou = false
    }

}


