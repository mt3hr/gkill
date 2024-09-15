'use strict';

import { Tag } from '@/classes/datas/tag';
import { GkillAPIResponse } from '../gkill-api-response';


export class GetTagsByTargetIDResponse extends GkillAPIResponse {


    tags: Array<Tag>;

    constructor() {
        super()
        this.tags = new Array<Tag>()
    }


}



