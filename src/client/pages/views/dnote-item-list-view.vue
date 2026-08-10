<template>
    <div class="dnote_item_list_root" @dragover="onListDragover" @drop="onListDrop">
        <DnoteItemView v-for="(dnote_item, index) in model_value" :key="dnote_item.id" v-model="model_value[index]"
            :editable="editable" :dnd_list_index="dnd_list_index" :application_config="application_config"
            :gkill_api="gkill_api"
            v-on="crudRelayHandlers"
            @requested_move_dnote_item="(list_id: string, list_index: number, item_id: string, item_index: number, direction: 'up' | 'down') => emits('requested_move_dnote_item', list_id, list_index, item_id, item_index, direction)"
            @requested_delete_dnote_item="(id: string) => delete_dnote_item(id)"
            @requested_update_dnote_item="(d: DnoteItemData) => update_dnote_item(d)"
            @finish_a_aggregate_task="emits('finish_a_aggregate_task')" ref="dnote_item_views" />
    </div>
</template>

<script lang="ts" setup>
import DnoteItemView from "./dnote-item-view.vue"
import type DnoteItemListViewProps from "./dnote-item-list-view-props"
import type DnoteItemListViewEmits from "./dnote-item-list-view-emits"
import type DnoteItem from "../../classes/dnote/dnote-item"
type DnoteItemData = DnoteItem
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
    onListDragover,
    onListDrop,
    crudRelayHandlers,
} = useDnoteItemListView({ props, emits, model_value })

defineExpose({ load_aggregated_value, reset })
</script>

<style scoped>
.dnote_item_list_root {
    min-height: 40px;
}
</style>
