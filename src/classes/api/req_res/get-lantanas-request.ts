'use strict';

import { FindLantanaQuery } from '../find_query/find-lantana-query';
import { GkillAPIRequest } from '../gkill-api-request';


export class GetLantanasRequest extends GkillAPIRequest {


    query: FindLantanaQuery;

    constructor() {
        super()
        this.query = new FindLantanaQuery()
    }


}



