'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { Kyou } from "@/classes/datas/kyou"
import type { Tag } from "@/classes/datas/tag"
import type { Text } from "@/classes/datas/text"
import type { Notification } from "@/classes/datas/notification"
import type { RudbeckiaPageKind } from "@/pages/views/rudbeckia-page-kind"

/**
 * ホストしている画面ビューの emit をそのまま上げる面。
 * rykv / mi / plaing / dashboard の4つに共通する17件 ＋ ポート固有の2件。
 */
export interface RudbeckiaPageDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'registered_kyou', registered_kyou: Kyou): void
    (e: 'updated_kyou', updated_kyou: Kyou): void
    (e: 'deleted_kyou', deleted_kyou: Kyou): void
    (e: 'registered_tag', registred_tag: Tag): void
    (e: 'updated_tag', updated_tag: Tag): void
    (e: 'deleted_tag', deleted_tag: Tag): void
    (e: 'registered_text', registered_text: Text): void
    (e: 'updated_text', updated_text: Text): void
    (e: 'deleted_text', deleted_text: Text): void
    (e: 'registered_notification', registered_notification: Notification): void
    (e: 'updated_notification', updated_notification: Notification): void
    (e: 'deleted_notification', deleted_notification: Notification): void
    (e: 'requested_show_application_config_dialog'): void
    (e: 'requested_reload_application_config'): void
    (e: 'saved_kyou_by_kftl', last_added_request_time: Date): void
    /** 中の画面切替メニューが選ばれた。ポートは「別のウィンドウを開く」に読み替える */
    (e: 'requested_open_page', kind: RudbeckiaPageKind): void
    /** 中の画面切替メニューでポートの対象外の画面が選ばれた。ページ遷移する */
    (e: 'requested_navigate_page', page_name: string): void
    /** ×・Escape・ブラウザバックのどれで閉じてもちょうど1回上がる */
    (e: 'closed'): void
}
