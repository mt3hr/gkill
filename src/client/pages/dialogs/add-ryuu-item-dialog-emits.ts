import type RelatedKyouQuery from "@/classes/dnote/related-kyou-query"
import type { GkillError } from "../../classes/api/gkill-error"
import type { GkillMessage } from "../../classes/api/gkill-message"

export default interface AddRyuuItemDialogEmits {
    (e: 'requested_add_related_kyou_query', related_kyou_query: RelatedKyouQuery): void
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
}