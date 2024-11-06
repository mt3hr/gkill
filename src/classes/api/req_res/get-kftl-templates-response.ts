'use strict'

import { KFTLTemplateElementData } from '@/classes/datas/kftl-template-element-data'
import { GkillAPIResponse } from '../gkill-api-response'

export class GetKFTLTemplatesResponse extends GkillAPIResponse {

    kftl_templates: Array<KFTLTemplateElementData>

    parsed_kftl_template_elements: KFTLTemplateElementData

    async parse_template(): Promise<void> {
        throw new Error('Not implemented')
    }

    constructor() {
        super()
        this.kftl_templates = new Array<KFTLTemplateElementData>()
        this.parsed_kftl_template_elements = new KFTLTemplateElementData()
    }

}


