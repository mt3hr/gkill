import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import type { DnoteCorrelationGraphQuery } from "@/classes/dnote/dnote-correlation"

export default interface DnoteCorrelationGraphEditorViewEmits {
    (e: 'saved', dnote_correlation_graph_query: DnoteCorrelationGraphQuery): void
    (e: 'received_messages', message: Array<GkillMessage>): void
    (e: 'received_errors', errors: Array<GkillError>): void
}
