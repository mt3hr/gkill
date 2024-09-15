'use strict';

export interface miShareFooterEmits {
    (e: 'request_open_share_mi_dialog'): void
    (e: 'request_open_manage_share_mi_dialog'): void
}
