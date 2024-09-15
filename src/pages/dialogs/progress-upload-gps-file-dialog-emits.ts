'use strict';

import type { GkillError } from "@/classes/api/gkill-error";
import type { GkillMessage } from "@/classes/api/gkill-message";
import type { Kyou } from "@/classes/datas/kyou";

export interface ProgressUploadGPSFileDialogEmits {
    (e: 'reveived_messages', message: Array<GkillMessage>): void 
    (e: 'received_errors', errors: Array<GkillError>): void 
    (e: 'requested_close_dialog',): void 
    (e: 'requested_show_decide_related_time_dialog', uploaded_kyous: Array<Kyou>): void 
}
