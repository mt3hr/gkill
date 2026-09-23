'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"

export interface GPSLogMapEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_focus_time', time: Date): void
    // 日付表示のピッカーで日を選んだ（その日の0:00）。地図に何日を映すかは親が持っているので、親が切り替える
    (e: 'requested_change_map_date', date: Date): void
}
