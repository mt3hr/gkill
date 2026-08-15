'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { Kyou } from "@/classes/datas/kyou"

export interface KFTLViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    // 送信で作った Kyou は他のAdd系と同じく1件ずつ registered_kyou で上げる。
    // 「終了」系は既存のTimeIsの更新なので updated_kyou
    (e: 'registered_kyou', kyou: Kyou): void
    (e: 'updated_kyou', kyou: Kyou): void
    // 作ったKyouを1件も引き直せなかったときのフォールバック
    (e: 'requested_reload_list'): void
    (e: 'saved_kyou_by_kftl', last_added_request_time: Date): void
}
