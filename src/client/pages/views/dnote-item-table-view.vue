<template>
    <div class="dnote_item_table_root">
        <table class="dnote_item_table">
            <tbody>
                <tr>
                    <td v-for="(list, listIndex) in model_value" :key="listIndex" class="dnote_item_table_td"
                        @dragover="onCellDragover" @drop="(e) => onCellDrop(e, listIndex)">
                        <DnoteItemListView v-model="model_value[listIndex]" :dnd_list_index="listIndex" :editable="editable"
                            :application_config="application_config" :gkill_api="gkill_api"
                            v-on="crudRelayHandlers"
                            @requested_move_dnote_item="(list_id: string, list_index: number, item_id: string, item_index: number, direction: 'up' | 'down') => handle_move_dnote_item(list_id, list_index, item_id, item_index, direction)"
                            @finish_a_aggregate_task="emits('finish_a_aggregate_task')"
                            ref="dnote_item_list_views" />
                    </td>
                </tr>
            </tbody>
        </table>
    </div>
</template>

<script lang="ts" setup>
import DnoteItemListView from "./dnote-item-list-view.vue"
import type DnoteItemTableViewEmits from "./dnote-item-table-view-emits"
import type DnoteItemTableViewProps from "./dnote-item-table-view-props"
import type DnoteItem from "@/classes/dnote/dnote-item"
import { useDnoteItemTableView } from '@/classes/use-dnote-item-table-view'

const props = defineProps<DnoteItemTableViewProps>()
const emits = defineEmits<DnoteItemTableViewEmits>()

const model_value = defineModel<Array<Array<DnoteItem>>>({ default: () => [] })

const {
    // Template refs
    dnote_item_list_views,

    // Methods used in template
    handle_move_dnote_item,
    onCellDragover,
    onCellDrop,

    // Exposed methods
    load_aggregated_value,
    reset,

    // Event relay objects
    crudRelayHandlers,
} = useDnoteItemTableView({ props, emits, model_value })

defineExpose({ load_aggregated_value, reset })
</script>

<style scoped>
.dnote_item_table_root {
    overflow-x: visible;
}

.dnote_item_table {
    border-collapse: collapse;
    border-spacing: 0;
}

.dnote_item_table_td {
    vertical-align: top;
    min-width: 210px;
    padding: 0px;
}
</style>
