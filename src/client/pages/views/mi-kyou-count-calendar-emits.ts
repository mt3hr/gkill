'use strict'

export interface MiKyouCountCalendarEmits {
    (e: 'requested_focus_time', time: Date): void
}
