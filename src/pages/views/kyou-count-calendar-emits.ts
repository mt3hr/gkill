'use strict'

export interface KyouCountCalendarEmits {
    (e: 'requested_focus_time', time: Date): void
}
