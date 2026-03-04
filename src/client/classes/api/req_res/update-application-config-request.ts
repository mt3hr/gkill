'use strict'

import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { GkillAPIRequest } from '../gkill-api-request'

export class UpdateApplicationConfigRequest extends GkillAPIRequest {

    application_config: ApplicationConfig

    constructor() {
        super()
        this.application_config = new ApplicationConfig()
    }

}


