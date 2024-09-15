'use strict';

import type { GkillAPIResponse } from '@/classes/api/gkill-api-response';
import { KFTLRequest } from '../kftl-request';
import type { KFTLStatementLineContext } from '../kftl-statement-line-context';


export class KFTLPrototypeRequest extends KFTLRequest {


    constructor(request_id: string, context: KFTLStatementLineContext) {
        super(request_id, context)
    }

    async is_prototype_request(request: KFTLRequest): Promise<void> {
        throw new Error('Not implemented');
    }

    async do_request(): Promise<Array<GkillAPIResponse>> {
        throw new Error('Not implemented');
    }


}



