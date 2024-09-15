'use strict';

export interface RepQueryEmits {
    (e: 'request_clear_rep_query'): void
    (e: 'request_update_checked_reps', checked_reps: Array<string>): void
}
