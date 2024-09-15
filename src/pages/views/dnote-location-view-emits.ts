'use strict';

import type { GkillError } from "@/classes/api/gkill-error";
import type { GkillMessage } from "@/classes/api/gkill-message";
import type { Kmemo } from "@/classes/datas/kmemo";
import type { Kyou } from "@/classes/datas/kyou";
import type { Lantana } from "@/classes/datas/lantana";
import type { Mi } from "@/classes/datas/mi";
import type { Nlog } from "@/classes/datas/nlog";
import type { ReKyou } from "@/classes/datas/re-kyou";
import type { Tag } from "@/classes/datas/tag";
import type { TimeIs } from "@/classes/datas/time-is";
import type { URLog } from "@/classes/datas/ur-log";

export interface DnoteLocationViewEmits {
    (e: 'reveived_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'registered_kmemo', kmemo: Kmemo): void
    (e: 'registered_lantana', lantana: Lantana): void
    (e: 'registered_mi', mi: Mi): void
    (e: 'registered_nlog', nlog: Nlog): void
    (e: 'registered_tag', tag: Tag): void
    (e: 'registered_text', text: Text): void
    (e: 'registered_timeis', timeis: TimeIs): void
    (e: 'registered_urlog', urlog: URLog): void
    (e: 'registered_re_kyou', rekyou: ReKyou): void
    (e: 'updated_kmemo', kmemo: Kmemo): void
    (e: 'updated_lantana', lantana: Lantana): void
    (e: 'updated_mi', mi: Mi): void
    (e: 'updated_nlog', nlog: Nlog): void
    (e: 'updated_tag', tag: Tag): void
    (e: 'updated_text', text: Text): void
    (e: 'updated_timeis', timeis: TimeIs): void
    (e: 'updated_urlog', urlog: URLog): void
    (e: 'updated_idf_kyou', kyou: Kyou): void
    (e: 'updated_re_kyou', rekyou: ReKyou): void
    (e: 'deleted_kmemo', kmemo: Kmemo): void
    (e: 'deleted_lantana', lantana: Lantana): void
    (e: 'deleted_mi', mi: Mi): void
    (e: 'deleted_nlog', nlog: Nlog): void
    (e: 'deleted_tag', tag: Tag): void
    (e: 'deleted_text', text: Text): void
    (e: 'deleted_timeis', timeis: TimeIs): void
    (e: 'deleted_urlog', urlog: URLog): void
    (e: 'deleted_idf_kyou', kyou: Kyou): void
    (e: 'deleted_rekyou', rekyou: ReKyou): void
    (e: 'requested_reload_kyou', kyou: Kyou): void
    (e: 'requested_reload_list',): void
}