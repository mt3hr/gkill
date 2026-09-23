'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { ApplicationConfig } from "@/classes/datas/config/application-config"

export interface EditSavedFindQueryDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_apply_saved_find_query_struct', saved_find_query_data: Record<string, unknown>): void
    // 「検索条件」ダイアログの実行中セクション・ダッシュボードセクション（触ったセクションだけ出る）
    (e: 'requested_apply_playing_timeis', playing_timeis_data: Record<string, unknown>): void
    (e: 'requested_apply_dashboard_struct', dashboard_data: Record<string, unknown>): void
    (e: 'requested_reload_application_config', application_config: ApplicationConfig): void
}
