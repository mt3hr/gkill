'use strict';

import type { GkillAPIResponse } from '@/classes/api/gkill-api-response';
import { KFTLRequest } from '../kftl-request';
import type { KFTLStatementLineContext } from '../kftl-statement-line-context';


export class KFTLLantanaRequest extends KFTLRequest {


    private mood: Number;

    constructor(request_id: string, context: KFTLStatementLineContext) {
        super(request_id, context)
        this.mood = 0;

    }

    async do_request(): Promise<Array<GkillAPIResponse>> {
        throw new Error('Not implemented');
    }

    async set_mood(mood: Number): Promise<void> {
        throw new Error('Not implemented');
    }

    async parse_mood(mood: string): Promise<Number> {
        throw new Error('Not implemented');
    }


}



