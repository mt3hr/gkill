'use strict'

import { ApplicationConfig } from '@/classes/datas/config/application-config'
import { GkillAPIResponse } from '../gkill-api-response'

export class UpdateKFTLTemplateResponse extends GkillAPIResponse {

    application_config: ApplicationConfig

    constructor() {
        super()
        this.application_config = new ApplicationConfig()
    }

}


