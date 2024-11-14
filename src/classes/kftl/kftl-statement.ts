'use strict'

import { GkillAPI } from "../api/gkill-api"
import type { KFTLRequest } from "./kftl-request"
import { KFTLRequestMap } from "./kftl-request-map"
import type { KFTLStatementLine } from "./kftl-statement-line"
import { KFTLStatementLineConstructorFactory } from "./kftl-statement-line-constructor-factory"
import { KFTLStatementLineContext } from "./kftl-statement-line-context"
import { LineLabelData } from "./line-label-data"
import type { TextAreaInfo } from "./text-area-info"

export class KFTLStatement {

    private statement_text: string
    public static readonly lookahead_line_count = 50

    constructor(text: string) {
        this.statement_text = text
    }

    get_statement_text(): string {
        return this.statement_text
    }

    async generate_requests(): Promise<Array<KFTLRequest>> {
        const requests = new Array<KFTLRequest>()
        const lines = this.generate_kftl_lines()
        const map = new KFTLRequestMap()
        for (let i = 0; i < lines.length; i++) {
            const line = lines[i]
            await line.apply_this_line_to_request_map(map)
        }
        map.forEach(request => {
            requests.push(request)
        });
        return requests
    }

    generate_line_label_data(text_area_info: TextAreaInfo): Array<LineLabelData> {
        const label_datas = new Array<LineLabelData>()
        const lines = this.generate_kftl_lines()
        let prev_context: KFTLStatementLineContext | null = null
        for (let i = 0; i < lines.length; i++) {
            const line = lines[i]
            const label_data = new LineLabelData()
            label_data.lines = line.get_count_line_in_textarea(text_area_info)
            label_data.label = line.get_label_name(line.get_context())
            label_data.target_request_id = line.get_context().get_this_statement_line_target_id()
            label_datas.push(label_data)
            prev_context = line.get_context()
        }
        for (let cnt = 0, line = lines[lines.length - 1]; cnt < KFTLStatement.lookahead_line_count && line.get_context().get_next_statement_line_constructor() != null; cnt++) {
            const line_text = ""
            const next_line_text = ""
            const target_id: string = (prev_context != null && prev_context.get_next_statement_line_target_id() != null) ? prev_context.get_next_statement_line_target_id()!! : GkillAPI.get_instance().generate_uuid()!!
            const context = new KFTLStatementLineContext(line_text, next_line_text, target_id, lines, false)
            line = line.get_context().get_next_statement_line_constructor()!!(context.get_this_statement_line_target_id(), context)
            const label_data = new LineLabelData()
            label_data.lines = line.get_count_line_in_textarea(text_area_info)
            label_data.label = line.get_label_name(line.get_context())
            label_datas.push(label_data)
            lines.push(line)
        }
        return label_datas
    }

    private generate_kftl_line(context: KFTLStatementLineContext): KFTLStatementLine {
        const lines = context.get_kftl_statement_lines()
        if (0 < lines.length) {
            const prev_line = lines[lines.length - 1]
            if (prev_line != null && prev_line.get_context().get_next_statement_line_constructor() != null) {
                const this_line_constructor = prev_line.get_context().get_next_statement_line_constructor()
                if (this_line_constructor != null) {
                    const line = this_line_constructor(context.get_this_statement_line_text(), context)
                    return line
                }
            }
        }

        const line_text_constructor_fuction = KFTLStatementLineConstructorFactory.get_instance().generate_kmemo_constructor(context.get_this_statement_line_text())
        const line = line_text_constructor_fuction(context.get_this_statement_line_text(), context)
        return line
    }

    private generate_kftl_lines(): Array<KFTLStatementLine> {
        throw new Error('Not implemented')
    }

}


