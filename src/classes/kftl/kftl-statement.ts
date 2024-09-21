'use strict'

import type { KFTLRequest } from "./kftl-request"
import type { KFTLStatementLine } from "./kftl-statement-line"
import type { KFTLStatementLineContext } from "./kftl-statement-line-context"
import type { LineLabelData } from "./line-label-data"
import type { TextAreaInfo } from "./text-area-info"

export class KFTLStatement {

    private statement_text: string

    constructor(text: string) {
        this.statement_text = text
    }

    async get_statement_text(): Promise<string> {
        throw new Error('Not implemented')
    }

    async generate_requests(): Promise<Array<KFTLRequest>> {
        throw new Error('Not implemented')
    }

    async generate_line_label_data(text_area_info: TextAreaInfo): Promise<Array<LineLabelData>> {
        throw new Error('Not implemented')
    }

    private async generate_kftl_line(context: KFTLStatementLineContext): Promise<KFTLStatementLine> {
        throw new Error('Not implemented')
    }

    private async generate_kftl_lines(): Promise<Array<KFTLStatementLine>> {
        throw new Error('Not implemented')
    }

}


