// ˅
'use strict';

import { KFTLTemplateStruct } from '@/classes/datas/config/kftl-template-struct';
import { GkillAPIResponse } from '../gkill-api-response';
import { KFTLTemplateElement } from '@/classes/datas/kftl-template-element';

// ˄

export class GetKFTLTemplatesResponse extends GkillAPIResponse {
    // ˅
    
    // ˄

    kftl_templates: Array<KFTLTemplateStruct>;

    parsed_kftl_template_elements: Array<KFTLTemplateElement>;

    async parse_template(): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    constructor() {
        // ˅
        super()
        this.kftl_templates = new Array<KFTLTemplateStruct>()
        this.parsed_kftl_template_elements = new Array<KFTLTemplateElement>
        // ˄
    }

    // ˅
    
    // ˄
}

// ˅

// ˄
