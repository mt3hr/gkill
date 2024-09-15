'use strict';

import { GkillAPIResponse } from '../gkill-api-response';


export class GetAllTagNamesResponse extends GkillAPIResponse {


    tag_names: Array<string>;

    constructor() {
        super()
        this.tag_names = new Array<string>()
    }


}



