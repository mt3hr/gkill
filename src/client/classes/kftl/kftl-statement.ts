'use strict'

import { GkillAPI } from "../api/gkill-api"
import type { KFTLStatementLine } from "./kftl-statement-line"
import { KFTLStatementLineConstructorFactory } from "./kftl-statement-line-constructor-factory"
import { KFTLStatementLineContext } from "./kftl-statement-line-context"
import { LineLabelData } from "./line-label-data"
import { is_save_charactor_line } from "./kftl-prefixes"
import type { TextAreaInfo } from "./text-area-info"

/**
 * メモ帳（KFTL）の本文を行に分類して、行ラベルを作る。
 *
 * **ここは表示（行ラベル）のためだけの分類器。** 「どの行が正しいか」「何を書くか」は
 * サーバの Go 実装（`src/server/gkill/api/kftl/`）だけが持ち、Web は
 * `/api/parse_kftl_text` の結果でピンクを塗り、`/api/submit_kftl_text` で書く（ADR-0507）。
 * 2026-09-15 までは TS 側にも解釈と `add_*` への fan-out があり、Go だけに入った修正が
 * Web に届かないまま残る（`/mood` 単独で気分0が書かれる等）事故を繰り返していた。
 */
export class KFTLStatement {

    private statement_text: string
    public static readonly lookahead_line_count = 50

    constructor(text: string) {
        this.statement_text = text
    }

    get_statement_text(): string {
        return this.statement_text
    }

    generate_line_label_data(text_area_info: TextAreaInfo): Array<LineLabelData> {
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
        // 先読み: まだ書いていない行のラベルも、次の行のコンストラクタがある限り上限まで組み立てる
        // （`/mi` と打った時点で「タイトル↓」「板名↓」… が見える）
        for (let cnt = 0, line = lines[lines.length - 1]; cnt < KFTLStatement.lookahead_line_count && line.get_context().get_next_statement_line_constructor() != null; cnt++) {
            const line_text = ""
            const next_line_text = ""
            const target_id: string = (prev_context != null && prev_context.get_next_statement_line_target_id() != null) ? prev_context.get_next_statement_line_target_id()! : GkillAPI.get_gkill_api().generate_uuid()!
            const context = new KFTLStatementLineContext(line_text, target_id, next_line_text, lines, false)
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

    /**
     * 本文を行に分類する。行の種別はサーバの Go 実装と同じ接頭辞規則で決める
     * （`kftl-prefixes.ts`）。ラベル用なので保存マーカーで止めない呼び方もできる
     */
    generate_kftl_lines(break_on_submit_marker: boolean = true): Array<KFTLStatementLine> {
        KFTLStatementLineConstructorFactory.get_instance().reset()
        const lines = new Array<KFTLStatementLine>()
        const text = this.get_statement_text()
        const line_texts = text.split("\n")
        let prev_context: KFTLStatementLineContext | null = null
        for (let i = 0; i < line_texts.length; i++) {
            const line_text = line_texts[i]
            const next_line_text = i < line_texts.length - 1 ? line_texts[i + 1] : ""
            const target_id: string = (prev_context != null && prev_context.get_next_statement_line_target_id() != null) ? prev_context.get_next_statement_line_target_id()! : GkillAPI.get_gkill_api().generate_uuid()
            const prototype_flag: boolean = (prev_context != null && prev_context.is_this_prototype() != null) ? prev_context?.is_next_prototype() : true
            const context: KFTLStatementLineContext = new KFTLStatementLineContext(line_text, target_id, next_line_text, lines.slice(0, i), prototype_flag)

            const line = this.generate_kftl_line(context)
            prev_context = context

            if (break_on_submit_marker && i != 0 && is_save_charactor_line(line_text)) {
                break
            }
            lines.push(line)
        }
        return lines
    }

}
