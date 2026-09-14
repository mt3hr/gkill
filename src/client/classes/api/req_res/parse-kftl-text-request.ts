'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

// メモ帳（KFTL）のテキストを解析だけする要求（Go 側 req_res.ParseKFTLTextRequest）。何も書かない。
// 打鍵のたびに投げて「おかしな行」を塗り、保存の直前に投げて未知タグ・未知板名の確認に使う（ADR-0507）。
export class ParseKFTLTextRequest extends GkillAPIRequest {

    kftl_text: string = ""

    constructor() {
        super()
    }

}
