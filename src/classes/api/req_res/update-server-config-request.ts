'use strict'

import { GkillAPIRequest } from '../gkill-api-request'
import { ServerConfig } from '@/classes/datas/config/server-config'

export class UpdateServerConfigRequest extends GkillAPIRequest {

    server_config: ServerConfig

    constructor() {
        super()
        this.server_config = new ServerConfig()
    }

}


