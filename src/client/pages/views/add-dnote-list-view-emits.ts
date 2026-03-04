import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type DnoteListQuery from "@/pages/views/dnote-list-query"

export default interface AddDnoteListViewEmits {
    (e: 'requested_add_dnote_list_query', dnote_list_item: DnoteListQuery): void
    (e: 'requested_close_dialog'): void
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
}