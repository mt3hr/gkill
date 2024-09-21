'use strict'

import { GkillAPI } from '../api/gkill-api'
import type { KFTLRequestMap } from './kftl-request-map'
import { KFTLStatementLineContext } from './kftl-statement-line-context'
import type { TextAreaInfo } from './text-area-info'

export abstract class KFTLStatementLine {

    private statement_line_text: string

    private api: GkillAPI

    private context: KFTLStatementLineContext

    constructor(line_text: string, context: KFTLStatementLineContext) {
        this.api = new GkillAPI()
        this.statement_line_text = line_text
        this.context = context
    }

    async get_statement_line_text(): Promise<string> {
        throw new Error('Not implemented')
    }

    async get_context(): Promise<KFTLStatementLineContext> {
        throw new Error('Not implemented')
    }

    async get_count_in_textarea(textarea_info: TextAreaInfo): Promise<Number> {
        throw new Error('Not implemented')
    }

    protected async get_prev_line(): Promise<KFTLStatementLine> {
        throw new Error('Not implemented')
    }

    abstract apply_this_line_to_request_map(requet_map: KFTLRequestMap): Promise<void>

    abstract get_label_name(context: KFTLStatementLineContext): Promise<string>

    private static async get_text_width(text: any, font: any): Promise<Number> {
        throw new Error('Not implemented')
    }

    private static async get_css_style(element: any, prop: any): Promise<string> {
        throw new Error('Not implemented')
    }

    private static async get_canvas_font(element: any): Promise<string> {
        throw new Error('Not implemented')
    }

}


