'use strict'

export interface ConfirmLogoutViewEmits {
    (e: 'requested_logout', close_database: boolean): void
    (e: 'requested_close_dialog'): void
}
