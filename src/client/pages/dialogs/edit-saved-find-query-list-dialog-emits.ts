'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { SavedFindQueryItem } from "@/classes/datas/config/saved-find-query-config"

export interface EditSavedFindQueryListDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_apply_saved_find_querys', items: Array<SavedFindQueryItem>): void
}
