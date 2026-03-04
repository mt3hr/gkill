'use strict'

import { ServerConfig } from '@/classes/datas/config/server-config'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetServerConfigsResponse extends GkillAPIResponse {

    server_configs: Array<ServerConfig>

    constructor() {
        super()
        this.server_configs = new Array<ServerConfig>()
    }

}


