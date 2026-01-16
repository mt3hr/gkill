'use strict'

import { URLog } from '@/classes/datas/ur-log'
import { GkillAPIRequest } from '../gkill-api-request'
import { Kyou } from '@/classes/datas/kyou'

export class UpdateURLogRequest extends GkillAPIRequest {

    urlog: URLog

    re_get_urlog_content: boolean

    tx_id: string | null = null

    want_response_kyou: boolean

    updated_kyou: Kyou | null | null = null

    constructor() {
        super()
        this.re_get_urlog_content = true
        this.urlog = new URLog()
        this.want_response_kyou = false
    }

}


