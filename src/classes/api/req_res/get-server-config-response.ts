'use strict';

import { ServerConfig } from '@/classes/datas/config/server-config';
import { GkillAPIResponse } from '../gkill-api-response';


export class GetServerConfigResponse extends GkillAPIResponse {


    server_config: ServerConfig;

    constructor() {
        super()
        this.server_config = new ServerConfig()
    }


}



