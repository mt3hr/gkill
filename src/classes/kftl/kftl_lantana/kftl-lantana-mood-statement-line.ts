'use strict'

import type { KFTLRequestMap } from '../kftl-request-map'
import { KFTLStatementLine } from '../kftl-statement-line'
import type { KFTLStatementLineContext } from '../kftl-statement-line-context'

export class KFTLLantanaMoodStatementLine extends KFTLStatementLine {

    constructor(line_text: string, context: KFTLStatementLineContext) {
        super(line_text, context)
    }

    async apply_this_line_to_request_map(requet_map: KFTLRequestMap): Promise<void> {
        throw new Error('Not implemented')
    }

    async get_label_name(context: KFTLStatementLineContext): Promise<string> {
        throw new Error('Not implemented')
    }

}


