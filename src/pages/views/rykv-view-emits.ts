'use strict'

import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { Kyou } from "@/classes/datas/kyou"
import type { Tag } from "@/classes/datas/tag"

export interface rykvViewEmits {
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
    (e: 'registered_kyou', kyou: Kyou): void
    (e: 'updated_kyou', kyou: Kyou): void
    (e: 'deleted_kyou', kyou: Kyou): void
    (e: 'registered_tag', tag: Tag): void
    (e: 'updated_tag', tag: Tag): void
    (e: 'deleted_tag', tag: Tag): void
    (e: 'registered_text', text: Text): void
    (e: 'updated_text', text: Text): void
    (e: 'deleted_text', text: Text): void
}
