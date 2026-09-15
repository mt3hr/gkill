'use strict'

export class GkillError {

    error_code: string

    error_message: string

    // 誰の問題か（input / auth / permission / not_found / conflict / too_large / rate_limit / config / server）。
    // サーバがエラーコードから決めて返す。クライアント生成のエラーは生成側が入れる（空なら server 扱い）
    error_kind: string

    // 何が起きたか（write_rep_missing / storage_unavailable / db_busy / timeout ...）。
    // サーバが Go の error を分類して返す。分類できないときは空。ヒント文の選択で kind より優先される
    reason: string

    show_keep: boolean

    constructor() {
        this.error_code = ""
        this.error_message = ""
        this.error_kind = ""
        this.reason = ""
        this.show_keep = true
    }

}
