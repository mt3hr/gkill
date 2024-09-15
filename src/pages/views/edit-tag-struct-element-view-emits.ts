'use strict';

import type { GkillError } from "@/classes/api/gkill-error";
import type { GkillMessage } from "@/classes/api/gkill-message";
import type { TagStructElementData } from "@/classes/datas/config/tag-struct-element-data";

export interface EditTagStructElementViewEmits {
    (e: 'reveived_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_update_tag_struct_element', tag_struct_element: TagStructElementData): void
}
