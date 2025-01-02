'use strict'

import { Notification } from '@/classes/datas/notification'
import { GkillAPIRequest } from '../gkill-api-request'

export class AddNotificationRequest extends GkillAPIRequest {

    notification: Notification

    constructor() {
        super()
        this.notification = new Notification()
    }

}


