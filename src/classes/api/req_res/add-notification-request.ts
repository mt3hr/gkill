'use strict'

import { Notification } from '@/classes/datas/notification'
import { GkillAPIRequest } from '../gkill-api-request'

export class AddNotificationRequest extends GkillAPIRequest {

    notification: Notification

    tx_id: string | null = null

    constructor() {
        super()
        this.notification = new Notification()
    }

}


