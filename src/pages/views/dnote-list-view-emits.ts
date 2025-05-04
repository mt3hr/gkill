import type DnoteListQuery from "./dnote-list-query"
import type { KyouViewEmits } from "./kyou-view-emits"

export default interface DnoteListViewEmits extends KyouViewEmits {
    (e: 'requested_delete_dnote_list_query', dnote_list_query_id: string): void
    (e: 'requested_update_dnote_list_query', dnote_list_query: DnoteListQuery): void
}