'use strict'

import { Notification } from '@/classes/datas/notification'
import { GkillAPIResponse } from '../gkill-api-response'

export class AddNotificationResponse extends GkillAPIResponse {

    added_notification: Notification

    constructor() {
        super()
        this.added_notification = new Notification()
    }

}


