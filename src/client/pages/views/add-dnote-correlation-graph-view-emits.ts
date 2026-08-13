import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { DnoteCorrelationGraphQuery } from "@/classes/dnote/dnote-correlation"

export default interface AddDnoteCorrelationGraphViewEmits {
    (e: 'requested_add_dnote_correlation_graph', dnote_correlation_graph_query: DnoteCorrelationGraphQuery): void
    (e: 'requested_close_dialog'): void
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
}
