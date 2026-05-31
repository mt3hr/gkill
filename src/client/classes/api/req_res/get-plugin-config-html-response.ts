'use strict'

import { GkillError } from '../gkill-error'
import { GkillMessage } from '../gkill-message'

export class GetPluginConfigHTMLResponse {
    errors: Array<GkillError>
    messages: Array<GkillMessage>
    html: string

    constructor() {
        this.errors = new Array<GkillError>()
        this.messages = new Array<GkillMessage>()
        this.html = ''
    }
}
