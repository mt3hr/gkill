'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { Kyou } from "@/classes/datas/kyou"

export interface KFTLDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    // 送信で作った Kyou は他のAdd系と同じく1件ずつ registered_kyou で上げる。
    // 「終了」系は既存のTimeIsの更新なので updated_kyou
    (e: 'registered_kyou', kyou: Kyou): void
    (e: 'updated_kyou', kyou: Kyou): void
    // 作ったKyouを1件も引き直せなかったときのフォールバック
    (e: 'requested_reload_list'): void
    // KFTL はタグを registered_tag で上げてこないので、保存完了のこの合図で
    // 板ツリー/タグツリーを取り直す（受け側は useConfigStructSync の resync_structs）
    (e: 'saved_kyou_by_kftl', last_added_request_time: Date): void
    // ×・Escape・ブラウザバックのどれで閉じても1回だけ上がる。
    // ホスト（kftl-dialog-host.vue）が開いているウィンドウの一覧から取り除く
    (e: 'closed'): void
}
