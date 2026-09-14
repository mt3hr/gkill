'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

// メモ帳（KFTL）のテキストをサーバに解釈・記録させる要求（Go 側 req_res.SubmitKFTLTextRequest）。
// 解釈と書き込みはサーバの1実装だけで行う（ADR-0507）。Wear OS / MCP と同じ入口。
export class SubmitKFTLTextRequest extends GkillAPIRequest {

    kftl_text: string = ""

    // 同じ送信の再配送を1回の登録に畳むためのキー。送信のたびに新しい UUID を採る
    // （内容ハッシュにしない ―― 意図した同一内容の再送が畳まれて記録できなくなる）
    idempotency_key: string = ""

    constructor() {
        super()
    }

}
