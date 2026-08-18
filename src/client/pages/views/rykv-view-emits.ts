'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { Kyou } from "@/classes/datas/kyou"
import type { Tag } from "@/classes/datas/tag"

export interface RykvViewEmits {
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
    // KFTL/MKFL はタグを registered_tag で上げてこないので、保存完了のこの合図で
    // ページ側が板ツリー/タグツリーを取り直す（useConfigStructSync の resync_structs）
    (e: 'saved_kyou_by_kftl', last_added_request_time: Date): void
    // ツールバーの画面切替メニューが選ばれた。is_hosted_in_dialog のときだけ上がる。
    // 単独ページでは今までどおりビューの中で router.replace する ―― ポート(rudbeckia)では
    // ページ遷移せずに別のダイアログを開きたいので、行き先の決定を親へ委ねる
    (e: 'requested_navigate_page', page_name: string): void
}
