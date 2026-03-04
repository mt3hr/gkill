'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { KC } from "@/classes/datas/kc"
import type { Kmemo } from "@/classes/datas/kmemo"
import type { Lantana } from "@/classes/datas/lantana"
import type { Mi } from "@/classes/datas/mi"
import type { Nlog } from "@/classes/datas/nlog"
import type { Tag } from "@/classes/datas/tag"
import type { TimeIs } from "@/classes/datas/time-is"
import type { URLog } from "@/classes/datas/ur-log"

export interface KFTLDialogEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'registered_kmemo', kmemo: Kmemo): void
    (e: 'registered_kc', kc: KC): void
    (e: 'registered_lantana', lantana: Lantana): void
    (e: 'registered_mi', mi: Mi): void
    (e: 'registered_nlog', nlog: Nlog): void
    (e: 'registered_tag', tag: Tag): void
    (e: 'registered_text', text: Text): void
    (e: 'registered_timeis', timeis: TimeIs): void
    (e: 'registered_urlog', urlog: URLog): void
    (e: 'updated_timeis', timeis: TimeIs): void
}
