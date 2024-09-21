'use strict'

export interface miBoardQueryEmits {
    (e: 'request_open_focus_board', board_name: string): void
}
