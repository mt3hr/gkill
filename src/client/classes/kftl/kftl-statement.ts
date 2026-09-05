'use strict'

import { GkillAPI } from "../api/gkill-api"
import type { KFTLRequest } from "./kftl-request"
import { KFTLRequestMap } from "./kftl-request-map"
import type { KFTLStatementLine } from "./kftl-statement-line"
import { KFTLStatementLineConstructorFactory } from "./kftl-statement-line-constructor-factory"
import { KFTLStatementLineContext } from "./kftl-statement-line-context"
import { KFTLSplitAndNextSecondStatementLine } from "./kftl_split/kftl-split-and-next-second-statement-line"
import { LineLabelData } from "./line-label-data"
import { is_save_charactor_line } from "./kftl-prefixes"
import type { TextAreaInfo } from "./text-area-info"
import type { ApplicationConfig } from "../datas/config/application-config"
import { expand_repeats, validate_repeats } from "./kftl_repeat/kftl-repeat-expand"

export class KFTLStatement {

    private statement_text: string
    public static readonly lookahead_line_count = 50

    constructor(text: string) {
        this.statement_text = text
    }

    get_statement_text(): string {
        return this.statement_text
    }

    /**
     * 送信するリクエストを組み立てる。
     *
     * gkill_api / application_config は繰り返しの既存判定にだけ使う。
     * 渡さなければ既存判定を飛ばす（単体テストのようにAPIが無い場合）。
     */
    async generate_requests(gkill_api: GkillAPI | null = null, application_config: ApplicationConfig | null = null): Promise<Array<KFTLRequest>> {
        const base = new Date(Date.now())
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
        // 繰り返し（「？？」）の展開。**行の解釈が全部終わってから**やる。
        // ここでやると「？？」をブロックのどこに書いても結果が同じになり、
        // 打鍵のたびに走る get_invalid_line_indexs とも切り離せる
        return expand_repeats(requests, base, gkill_api, application_config)
    }

    generate_line_label_data(text_area_info: TextAreaInfo): Array<LineLabelData> {
        const tx_id = "" // label_data作るためには必要ない
        const label_datas = new Array<LineLabelData>()
        const lines = this.generate_kftl_lines(false)
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
            const target_id: string = (prev_context != null && prev_context.get_next_statement_line_target_id() != null) ? prev_context.get_next_statement_line_target_id()! : GkillAPI.get_gkill_api().generate_uuid()!
            const context = new KFTLStatementLineContext(tx_id, line_text, target_id, next_line_text, lines, false)
            line = line.get_context().get_next_statement_line_constructor()!(context.get_this_statement_line_text(), context)
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

    private generate_kftl_lines(break_on_submit_marker: boolean = true): Array<KFTLStatementLine> {
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
            const target_id: string = (prev_context != null && prev_context.get_next_statement_line_target_id() != null) ? prev_context.get_next_statement_line_target_id()! : GkillAPI.get_gkill_api().generate_uuid()
            const prototype_flag: boolean = (prev_context != null && prev_context.is_this_prototype() != null) ? prev_context?.is_next_prototype() : true
            const context: KFTLStatementLineContext = new KFTLStatementLineContext(tx_id, line_text, target_id, next_line_text, lines.slice(0, i), prototype_flag)
            context.set_add_second(prev_add_second)

            const line = this.generate_kftl_line(context)

            // クラス名ではなくコンストラクタの同一性で比較する。
            // minify するとクラス名は短縮され、別クラスが同じ名前になりうるため。
            if (line.constructor === KFTLSplitAndNextSecondStatementLine) {
                prev_add_second++
            }
            prev_context = context

            if (break_on_submit_marker && i != 0 && is_save_charactor_line(line_text)) {
                break
            }
            lines.push(line)
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
            } catch (_e: unknown) {
                invalid_line_indexs.push(i)
            }
        }
        // 繰り返しの検査。**ここでは複製しない** ―― この関数は本文が変わるたびに走るので、
        // 展開すると打鍵1回あたり最大1000件を作ることになる。
        // 行ごとの apply では見えない失敗（日時欄が無い・条件に合う日が無い・上限超過）を
        // ここで「？？」の開始行へ結びつける
        if (invalid_line_indexs.length === 0) {
            const requests = new Array<KFTLRequest>()
            map.forEach(request => {
                requests.push(request)
            });
            invalid_line_indexs.push(...validate_repeats(requests, new Date(Date.now())))
        }
        return invalid_line_indexs
    }

}


