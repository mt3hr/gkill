'use strict'

import { URLog } from '@/classes/datas/ur-log'
import { GkillAPIRequest } from '../gkill-api-request'

export class UpdateURLogRequest extends GkillAPIRequest {

    urlog: URLog
    re_get_urlog_content: boolean

    constructor() {
        super()
        this.re_get_urlog_content = true
        this.urlog = new URLog()
    }

}


