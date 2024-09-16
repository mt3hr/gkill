'use strict';

import type { GkillError } from "@/classes/api/gkill-error";
import type { GkillMessage } from "@/classes/api/gkill-message";
import type { Kmemo } from "@/classes/datas/kmemo";
import type { Lantana } from "@/classes/datas/lantana";
import type { Tag } from "@/classes/datas/tag";

export interface AddLantanaViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'registered_lantana', lantana: Lantana): void
    (e: 'registered_kmemo', kmemo: Kmemo): void
    (e: 'registered_tag', tag: Tag): void
    (e: 'registered_text', text: Text): void
}
