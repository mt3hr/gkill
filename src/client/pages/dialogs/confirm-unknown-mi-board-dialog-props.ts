'use strict'

export interface ConfirmUnknownMiBoardDialogProps {
    /** まだ実在しない板名。空なら表示しない */
    unknown_mi_boards: Array<string>
    /** 保存ボタンの二重押し防止 */
    is_requested_submit: boolean
}
