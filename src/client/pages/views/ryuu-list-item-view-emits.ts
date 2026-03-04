import type { KyouViewEmits } from "./kyou-view-emits";
import type RelatedKyouQuery from "../../classes/dnote/related-kyou-query";

export default interface RyuuListItemViewEmits extends KyouViewEmits {
    (e: 'requested_delete_related_kyou_list_query', related_kyou_list_query_id: string): void
    (e: 'requested_update_related_kyoudnote_list_query', related_kyou_list_query: RelatedKyouQuery): void
    (e: 'requested_move_related_kyou_query', srcId: string, targetId: string, dropType: 'up' | 'down'): void
}