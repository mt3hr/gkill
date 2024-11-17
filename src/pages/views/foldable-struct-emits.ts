'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { CheckState } from "./check-state"
import type { DropTypeFoldableStruct } from "@/classes/api/drop-type-foldable-struct"
import type { FoldableStructModel } from "./foldable-struct-model"

export interface FoldableStructEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'clicked_items', items: Array<string>, check_state: CheckState, is_by_user: boolean): void
    (e: 'dblclicked_item', id: string): void
    (e: 'requested_update_check_state', items: Array<string>, check_state: CheckState): void
    (e: 'requested_move_struct_obj', struct_obj: FoldableStructModel, target_struct_obj: FoldableStructModel, drop_type: DropTypeFoldableStruct): void
    (e: 'requested_update_struct_obj', struct_obj: FoldableStructModel): void
}
