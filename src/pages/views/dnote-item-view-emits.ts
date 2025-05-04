import type DnoteItem from "@/classes/dnote/dnote-item";
import type { KyouViewEmits } from "./kyou-view-emits";

export default interface DnoteItemViewEmits extends KyouViewEmits {
    (e: 'requested_delete_dnote_item', dnote_item_id: string): void
    (e: 'requested_update_dnote_item', dnote_item: DnoteItem): void
}