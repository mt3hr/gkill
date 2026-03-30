<template>
    <div class="dnote_item_list_root" @dragover="on_list_dragover" @drop="on_list_drop">
        <DnoteItemView v-for="(dnote_item, index) in model_value" :key="dnote_item.id" v-model="model_value[index]"
            :editable="editable" :dnd_list_index="dnd_list_index" :application_config="application_config"
            :gkill_api="gkill_api"
            @requested_move_dnote_item="(list_id: string, list_index: number, item_id: string, item_index: number, direction: 'up' | 'down') => emits('requested_move_dnote_item', list_id, list_index, item_id, item_index, direction)"
            @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
            @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
            @focused_kyou="(kyou: Kyou) => emits('focused_kyou', kyou)"
            @clicked_kyou="(kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) }"
            @requested_delete_dnote_item="(id: string) => delete_dnote_item(id)"
            @requested_update_dnote_item="(d: DnoteItemData) => update_dnote_item(d)"
            @deleted_kyou="(kyou: Kyou) => emits('deleted_kyou', kyou)"
            @deleted_tag="(tag: Tag) => emits('deleted_tag', tag)"
            @deleted_text="(text: Text) => emits('deleted_text', text)"
            @deleted_notification="(n: Notification) => emits('deleted_notification', n)"
            @registered_kyou="(kyou: Kyou) => emits('registered_kyou', kyou)"
            @registered_tag="(tag: Tag) => emits('registered_tag', tag)"
            @registered_text="(text: Text) => emits('registered_text', text)"
            @registered_notification="(n: Notification) => emits('registered_notification', n)"
            @updated_kyou="(kyou: Kyou) => emits('updated_kyou', kyou)"
            @updated_tag="(tag: Tag) => emits('updated_tag', tag)"
            @updated_text="(text: Text) => emits('updated_text', text)"
            @updated_notification="(n: Notification) => emits('updated_notification', n)"
            @requested_open_rykv_dialog="(kind: RykvDialogKind, kyou: Kyou, payload?: RykvDialogPayload) => emits('requested_open_rykv_dialog', kind, kyou, payload)"
            @finish_a_aggregate_task="emits('finish_a_aggregate_task')" ref="dnote_item_views" />
    </div>
</template>

<script lang="ts" setup>
import DnoteItemView from "./dnote-item-view.vue"
import type { RykvDialogKind, RykvDialogPayload } from "./rykv-dialog-kind"
import type DnoteItemListViewProps from "./dnote-item-list-view-props"
import type DnoteItemListViewEmits from "./dnote-item-list-view-emits"
import type DnoteItem from "../../classes/dnote/dnote-item"
type DnoteItemData = DnoteItem
import type { Kyou } from "@/classes/datas/kyou"
import type { Tag } from "@/classes/datas/tag"
import type { Text } from "@/classes/datas/text"
import type { Notification } from "@/classes/datas/notification"
import type { GkillError } from "@/classes/api/gkill-error"
import type { GkillMessage } from "@/classes/api/gkill-message"
import { useDnoteItemListView } from '@/classes/use-dnote-item-list-view'

const props = defineProps<DnoteItemListViewProps>()
const emits = defineEmits<DnoteItemListViewEmits>()
const model_value = defineModel<Array<DnoteItem>>({ default: () => [] })

const {
    dnote_item_views,
    dnd_list_index,
    load_aggregated_value,
    delete_dnote_item,
    update_dnote_item,
    reset,
    on_list_dragover,
    on_list_drop,
} = useDnoteItemListView({ props, emits, model_value })

defineExpose({ load_aggregated_value, reset })
</script>

<style scoped>
.dnote_item_list_root {
    min-height: 40px;
}
</style>
