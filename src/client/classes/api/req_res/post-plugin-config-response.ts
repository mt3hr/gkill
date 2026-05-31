'use strict'

import { GkillError } from '../gkill-error'
import { GkillMessage } from '../gkill-message'

export class PostPluginConfigResponse {
    errors: Array<GkillError>
    messages: Array<GkillMessage>

    constructor() {
        this.errors = new Array<GkillError>()
        this.messages = new Array<GkillMessage>()
    }
}
