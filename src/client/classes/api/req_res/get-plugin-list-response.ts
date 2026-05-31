'use strict'

import { GkillError } from '../gkill-error'
import { GkillMessage } from '../gkill-message'

export class PluginInfo {
    name: string
    version: string
    description: string
    data_type: string
    rep_name: string
    is_alive: boolean

    constructor() {
        this.name = ''
        this.version = ''
        this.description = ''
        this.data_type = ''
        this.rep_name = ''
        this.is_alive = false
    }
}

export class GetPluginListResponse {
    errors: Array<GkillError>
    messages: Array<GkillMessage>
    plugins: Array<PluginInfo>

    constructor() {
        this.errors = new Array<GkillError>()
        this.messages = new Array<GkillMessage>()
        this.plugins = new Array<PluginInfo>()
    }
}
