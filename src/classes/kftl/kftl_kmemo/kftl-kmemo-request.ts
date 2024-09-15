'use strict';

import type { GkillAPIResponse } from '@/classes/api/gkill-api-response';
import { KFTLRequest } from '../kftl-request';
import type { KFTLStatementLineContext } from '../kftl-statement-line-context';


export class KFTLKmemoRequest extends KFTLRequest {


    private kmemo_content: string;

    constructor(request_id: string, context: KFTLStatementLineContext) {
        super(request_id, context)
        this.kmemo_content = "";

    }

    async do_request(): Promise<Array<GkillAPIResponse>> {
        throw new Error('Not implemented');
    }

    async add_kmemo_line(kmemo_line: string): Promise<void> {
        throw new Error('Not implemented');
    }


}



