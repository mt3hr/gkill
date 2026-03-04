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
        this.api = GkillAPI.get_gkill_api()
        this.statement_line_text = line_text
        this.context = context
    }

    get_statement_line_text(): string {
        return this.statement_line_text
    }

    get_context(): KFTLStatementLineContext {
        return this.context
    }

    get_count_line_in_textarea(textarea_info: TextAreaInfo): Number {
        const textarea_element = document.getElementById(textarea_info.text_area_element_id)!!
        const kftl_text_area_width = textarea_element.clientWidth
        const text_width = KFTLStatementLine.get_text_width(this.get_statement_line_text(), KFTLStatementLine.get_canvas_font(textarea_element)).valueOf()
        const lines = 1 + parseInt(`${text_width / kftl_text_area_width}`)
        return lines
    }

    protected get_prev_line(): KFTLStatementLine | null {
        const statement_lines = this.get_context().get_kftl_statement_lines()
        if (1 <= statement_lines.length) {
            return statement_lines[statement_lines.length - 1]
        }
        return null
    }

    abstract apply_this_line_to_request_map(request_map: KFTLRequestMap): Promise<void>

    abstract get_label_name(context: KFTLStatementLineContext): string

    private static get_text_width(text: any, font: any): Number {
        const canvas: any = (KFTLStatementLine.get_text_width as any).canvas || ((KFTLStatementLine.get_text_width as any).canvas = document.createElement("canvas"))
        const context = canvas.getContext("2d")
        context.font = font
        const metrics = context.measureText(text)
        return metrics.width
    }

    private static get_css_style(element: any, prop: any): string {
        return window.getComputedStyle(element, null).getPropertyValue(prop)
    }

    private static get_canvas_font(element = document.body): string {
        const fontWeight = KFTLStatementLine.get_css_style(element, 'font-weight') || 'normal'
        const fontSize = KFTLStatementLine.get_css_style(element, 'font-size') || '16px'
        const fontFamily = KFTLStatementLine.get_css_style(element, 'font-family') || 'Times New Roman'
        return `${fontWeight} ${fontSize} ${fontFamily}`
    }
}


