'use strict'

import { GkillAPIResponse } from '../gkill-api-response'

// 連鎖削除で消した1件。data_type はエンティティ名（kmemo / mirekyou / tag / text / notification …）
export interface DeletedRecord {
    id: string
    data_type: string
}

export class DeleteKyouCascadeResponse extends GkillAPIResponse {

    // 消したもの全部（起点・付随データ・参照元）。失敗したときは空で、何も消えていない
    deleted: Array<DeletedRecord> = []

    constructor() {
        super()
    }

}
