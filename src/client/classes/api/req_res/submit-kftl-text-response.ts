'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

// サーバの KFTL が実際に書いた1件（Go 側 req_res.SubmitKFTLTextCreated）。
export interface SubmitKFTLTextCreated {
    id: string
    data_type: string
    // 新規作成ではなく既存レコードの更新（打刻の終了）
    updated: boolean
    // 書いた記録の関連時刻（打刻の終了は終了時刻）。JSON なので文字列で届く
    related_time: string
}

export class SubmitKFTLTextResponse extends GkillAPIResponse {

    // 実際に書いた記録。失敗したときは何も残らず空（commit は1つの SQLite トランザクション。ADR-0219）。
    // 冪等キーで再送を畳んだときも実行していないので空
    created: Array<SubmitKFTLTextCreated> | null = null

    constructor() {
        super()
    }

}
