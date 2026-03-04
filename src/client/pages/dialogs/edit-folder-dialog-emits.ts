'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { FolderStructElementData } from "@/classes/datas/config/folder-struct-element-data"

export interface EditFolderDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_update_folder', folder_struct_element: FolderStructElementData): void
}
