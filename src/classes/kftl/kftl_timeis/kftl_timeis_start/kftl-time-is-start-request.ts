// ˅
'use strict';

import type { GkillAPIResponse } from '@/classes/api/gkill-api-response';
import { KFTLRequest } from '../../kftl-request';
import type { KFTLRequestMap } from '../../kftl-request-map';
import type { KFTLStatementLineContext } from '../../kftl-statement-line-context';

// ˄

export class KFTLTimeIsStartRequest extends KFTLRequest {
    // ˅
    
    // ˄

    private title: string;

    constructor(line_text: string, context: KFTLStatementLineContext) {
        // ˅
        super(line_text, context)
        this.title = ""
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
