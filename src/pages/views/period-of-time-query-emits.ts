'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"

export interface PeriodOfTimeQueryEmits {
    (e: 'request_clear_use_period_of_time_query'): void
    (e: 'request_update_use_period_of_time', use_period_of_time: boolean): void
    (e: 'request_update_period_of_time', period_of_end_start_time_second: number | null, period_of_time_end_time_second: number | null, period_of_time_week_of_days: Array<number>): void
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'inited'): void
}
