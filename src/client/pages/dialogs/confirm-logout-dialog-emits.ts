'use strict'

export interface ConfirmLogoutDialogEmits {
    (e: 'requested_logout', close_database: boolean): void
}
