'use strict'

import { GkillAPIRequest } from '../gkill-api-request'
import { ServerConfig } from '@/classes/datas/config/server-config'

export class UpdateServerConfigsRequest extends GkillAPIRequest {

    server_configs: Array<ServerConfig>

    constructor() {
        super()
        this.server_configs = new Array<ServerConfig>()
    }

}


