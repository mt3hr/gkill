'use strict'

import type { KFTLRequestMap } from '@/classes/kftl/kftl-request-map'
import type { KFTLStatementLineContext } from '@/classes/kftl/kftl-statement-line-context'
import { KFTLStatementLine } from '../../../kftl-statement-line'

export class KFTLTimeIsEndByTagIfExistTagNameStatementLine extends KFTLStatementLine {

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


