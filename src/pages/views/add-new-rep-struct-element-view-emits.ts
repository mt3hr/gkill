'use strict';

import type { GkillError } from "@/classes/api/gkill-error";
import type { GkillMessage } from "@/classes/api/gkill-message";
import type { RepStructElementData } from "@/classes/datas/config/rep-struct-element-data";

export interface AddNewRepStructElementViewEmits {
    (e: 'reveived_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_add_rep_struct_element', rep_struct_element: RepStructElementData): void
}