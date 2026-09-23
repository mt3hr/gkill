'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

// 「おかしな行」1つ（Go 側 req_res.ParseKFTLTextInvalidLine）。
export interface ParseKFTLTextInvalidLine {
    // 1始まりの行番号。0 は「行が分からない」
    line_number: number
    line_text: string
    // ローカライズ済みの理由（submit_kftl_text の errors[].error_message と同じ文面）
    message: string
}

export class ParseKFTLTextResponse extends GkillAPIResponse {

    // 書き間違い。解析そのものは成功なので errors ではなくここに載る（空なら送信してよい）
    invalid_lines: Array<ParseKFTLTextInvalidLine> = []

    // 送信すると付くタグ名（重複なし・出現順）
    tags: Array<string> = []

    // 記録ごとのタグの組（組の中は重複なし・出現順、記録の登録順。タグの無い記録は入れない）。
    // 保存に成功したら組ごとにタグ履歴へ積む
    tag_groups: Array<Array<string>> = []

    // Mi / MiReKyou に書かれた板名（空欄は含めない・既定板へは解決しない）
    mi_board_names: Array<string> = []

    // 繰り返しを展開したあとの、書き込みの候補になる件数
    record_count: number = 0

    constructor() {
        super()
    }

}
