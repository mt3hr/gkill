'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"

/**
 * 出力 HTML の rykv（`exported-rykv-view.vue`）がページへ上げるもの。
 *
 * 出力 HTML は読むだけなので、上げるのはエラーとメッセージだけ。
 * mi の `exported-mi-view.vue` も同じものを上げる（`ExportedMiViewEmits`）。
 */
export interface ExportedRykvViewEmits {
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'received_messages', messages: Array<GkillMessage>): void
}
