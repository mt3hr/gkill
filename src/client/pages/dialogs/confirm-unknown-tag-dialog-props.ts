'use strict'

export interface ConfirmUnknownTagDialogProps {
    /** タグツリーに無いタグ名。空なら表示しない */
    unknown_tags: Array<string>
    /** 保存ボタンの二重押し防止 */
    is_requested_submit: boolean
}
