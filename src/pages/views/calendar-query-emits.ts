'use strict'

export interface CalendarQueryEmits {
    (e: 'request_clear_calendar_query'): void
    (e: 'request_update_use_calendar_query', use_calendar_query: boolean): void
    (e: 'request_update_dates', date1: Date|null, date2: Date|null): void
}
