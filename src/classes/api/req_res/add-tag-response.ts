'use strict';

import { Tag } from '@/classes/datas/tag';
import { GkillAPIResponse } from '../gkill-api-response';


export class AddTagResponse extends GkillAPIResponse {


    added_tag: Tag;

    constructor() {
        super()
        this.added_tag = new Tag()
    }


}



