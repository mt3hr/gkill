'use strict';

import type { GkillAPIResponse } from '@/classes/api/gkill-api-response';
import { KFTLRequest } from '../kftl-request';
import type { KFTLStatementLineContext } from '../kftl-statement-line-context';


export class KFTLMiRequest extends KFTLRequest {


    private title: string;

    private board_name: string;

    private limit_time: Date;

    private estimate_start_time: Date;

    private esitimate_end_time: Date;

    constructor(request_id: string, context: KFTLStatementLineContext) {
        super(request_id, context)
        this.title = ""
        this.board_name = ""
        this.limit_time = new Date(0)
        this.estimate_start_time = new Date(0)
        this.esitimate_end_time = new Date(0)
    }

    async do_request(): Promise<Array<GkillAPIResponse>> {
        throw new Error('Not implemented');
    }

    async set_title(title: string): Promise<void> {
        throw new Error('Not implemented');
    }

    async set_board_name(board_name: string): Promise<void> {
        throw new Error('Not implemented');
    }

    async set_limit_time(limit_time: Date): Promise<void> {
        throw new Error('Not implemented');
    }

    async set_estimate_start_time(estimate_start_time: Date): Promise<void> {
        throw new Error('Not implemented');
    }

    async set_estimate_end_time(estimate_end_time: Date): Promise<void> {
        throw new Error('Not implemented');
    }


}



