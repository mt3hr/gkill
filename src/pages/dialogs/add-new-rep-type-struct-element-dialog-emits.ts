'use strict';

import type { GkillError } from "@/classes/api/gkill-error";
import type { GkillMessage } from "@/classes/api/gkill-message";
import type { RepTypeStructElementData } from "@/classes/datas/config/rep-type-struct-element-data";

export interface AddNewRepTypeStructElementDialogEmits {
    (e: 'reveived_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'requested_add_rep_type_struct_element', rep_type_struct_element: RepTypeStructElementData): void
}