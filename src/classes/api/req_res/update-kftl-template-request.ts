'use strict'

import { GkillAPIRequest } from '../gkill-api-request'
import { KFTLTemplateStruct } from '../../datas/config/kftl-template-struct'

export class UpdateKFTLTemplateRequest extends GkillAPIRequest {

    kftl_templates: Array<KFTLTemplateStruct>

    constructor() {
        super()
    }

}


