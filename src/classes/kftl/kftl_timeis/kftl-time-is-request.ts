// ˅
'use strict';

import type { GkillAPIResponse } from '@/classes/api/gkill-api-response';
import { KFTLRequest } from '../kftl-request';
import type { KFTLRequestMap } from '../kftl-request-map';
import type { KFTLStatementLineContext } from '../kftl-statement-line-context';

// ˄

export class KFTLTimeIsRequest extends KFTLRequest {
    // ˅

    // ˄

    private title: string;

    private start_time: Date;

    private end_time: Date;

    constructor(line_text: string, context: KFTLStatementLineContext) {
        // ˅
        super(line_text, context)
        this.title = ""
        this.start_time = new Date(0)
        this.end_time = new Date(0)
        // ˄
    }

    async apply_this_line_to_request_map(requet_map: KFTLRequestMap): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async get_label_name(context: KFTLStatementLineContext): Promise<string> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async set_title(title: string): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async set_start_time(start_time: Date): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async set_end_time(end_time: Date): Promise<void> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    async do_request(): Promise<Array<GkillAPIResponse>> {
        // ˅
        throw new Error('Not implemented');
        // ˄
    }

    // ˅

    // ˄
}

// ˅

// ˄
