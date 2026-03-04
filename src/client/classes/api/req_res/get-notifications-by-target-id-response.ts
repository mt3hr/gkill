'use strict'

import { Notification } from '@/classes/datas/notification'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetNotificationsByTargetIDResponse extends GkillAPIResponse {

    notifications: Array<Notification>

    constructor() {
        super()
        this.notifications = new Array<Notification>()
    }

}


