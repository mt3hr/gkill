'use strict';

import type { GkillError } from "@/classes/api/gkill-error";
import type { GkillMessage } from "@/classes/api/gkill-message";
import type { Kyou } from "@/classes/datas/kyou";

export interface ProgressUploadGPSFileViewEmits {
    (e: 'reveived_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'uploaded_kyous', uploaded_kyous: Array<Kyou>): void
}
