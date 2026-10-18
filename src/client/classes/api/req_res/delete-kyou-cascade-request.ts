'use strict'

import { GkillAPIRequest } from '../gkill-api-request'

// 記録を連鎖して論理削除する要求（/api/delete_kyou_cascade。ADR-0636）。
// 付いているタグ・テキスト・通知と、参照しているリポスト・タスク化をサーバがまとめて消す。
export class DeleteKyouCascadeRequest extends GkillAPIRequest {

    // 起点の記録の ID
    target_id: string = ""

    // 起点の記録の種別。射影名（mi_create / mirekyou_create …）のままでよい
    data_type: string = ""

    // 削除の版に刻むアプリ名。空ならサーバが "gkill" にする
    update_app: string = "gkill"

    constructor() {
        super()
    }

}
