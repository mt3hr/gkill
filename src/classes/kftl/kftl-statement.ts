'use strict'

import { i18n } from "@/i18n"
import { GkillAPI } from "../api/gkill-api"
import type { KFTLRequest } from "./kftl-request"
import { KFTLRequestMap } from "./kftl-request-map"
import type { KFTLStatementLine } from "./kftl-statement-line"
import { KFTLStatementLineConstructorFactory } from "./kftl-statement-line-constructor-factory"
import { KFTLStatementLineContext } from "./kftl-statement-line-context"
import { KFTLSplitAndNextSecondStatementLine } from "./kftl_split/kftl-split-and-next-second-statement-line"
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
        const tx_id = "" // label_data作るためには必要ない
        const label_datas = new Array<LineLabelData>()
        const lines = this.generate_kftl_lines()
        let prev_context: KFTLStatementLineContext | null = null
        for (let i = 0; i < lines.length; i++) {
            const line = lines[i]
            const label_data = new LineLabelData()
            label_data.lines = line.get_count_line_in_textarea(text_area_info).valueOf()
            label_data.label = line.get_label_name(line.get_context())
            label_data.target_request_id = line.get_context().get_this_statement_line_target_id()
            label_datas.push(label_data)
            prev_context = line.get_context()
        }
        for (let cnt = 0, line = lines[lines.length - 1]; cnt < KFTLStatement.lookahead_line_count && line.get_context().get_next_statement_line_constructor() != null; cnt++) {
            const line_text = ""
            const next_line_text = ""
            const target_id: string = (prev_context != null && prev_context.get_next_statement_line_target_id() != null) ? prev_context.get_next_statement_line_target_id()!! : GkillAPI.get_gkill_api().generate_uuid()!!
            const context = new KFTLStatementLineContext(tx_id, line_text, next_line_text, target_id, lines, false)
            line = line.get_context().get_next_statement_line_constructor()!!(context.get_this_statement_line_target_id(), context)
            const label_data = new LineLabelData()
            label_data.lines = line.get_count_line_in_textarea(text_area_info).valueOf()
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
        const tx_id = GkillAPI.get_gkill_api().generate_uuid()
        KFTLStatementLineConstructorFactory.get_instance().reset()
        const lines = new Array<KFTLStatementLine>()
        const text = this.get_statement_text()
        const line_texts = text.split("\n")
        let prev_context: KFTLStatementLineContext | null = null
        let prev_add_second = 0
        for (let i = 0; i < line_texts.length; i++) {
            const line_text = line_texts[i]
            const next_line_text = i < line_texts.length - 1 ? line_texts[i + 1] : ""
            const target_id: string = (prev_context != null && prev_context.get_next_statement_line_target_id() != null) ? prev_context.get_next_statement_line_target_id()!! : GkillAPI.get_gkill_api().generate_uuid()
            const prototype_flag: boolean = (prev_context != null && prev_context.is_this_prototype() != null) ? prev_context?.is_next_prototype() : true
            const context: KFTLStatementLineContext = new KFTLStatementLineContext(tx_id, line_text, target_id, next_line_text, lines.slice(0, i), prototype_flag)
            context.set_add_second(prev_add_second)

            const line = this.generate_kftl_line(context)
            lines.push(line)

            if (line.constructor.name == KFTLSplitAndNextSecondStatementLine.name) {
                prev_add_second++
            }
            prev_context = context

            if (i != 0 && line_text == i18n.global.t("KFTL_SAVE_CHARACTOR")) {
                break
            }
        }
        return lines
    }

    public async get_invalid_line_indexs(): Promise<Array<number>> {
        const lines = this.generate_kftl_lines()
        const invalid_line_indexs = new Array<number>()
        const map = new KFTLRequestMap()
        for (let i = 0; i < lines.length; i++) {
            const line = lines[i]
            try {
                await line.apply_this_line_to_request_map(map)
            } catch (e: any) {
                invalid_line_indexs.push(i)
            }
        }
        return invalid_line_indexs
    }

}


