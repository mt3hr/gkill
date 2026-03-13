<template>
    <div class="dnote_item_table_root">
        <table class="dnote_item_table">
            <tr>
                <td v-for="(list, listIndex) in model_value" :key="listIndex" class="dnote_item_table_td"
                    @dragover="on_cell_dragover" @drop="(e) => on_cell_drop(e, listIndex)">
                    <DnoteItemListView v-model="model_value[listIndex]" :dnd_list_index="listIndex" :editable="editable"
                        :application_config="application_config" :gkill_api="gkill_api"
                        @requested_move_dnote_item="(...args: any[]) => handle_move_dnote_item(args[0] as string, args[1] as number, args[2] as string, args[3] as number, args[4] as 'up' | 'down')"
                        @finish_a_aggregate_task="emits('finish_a_aggregate_task')"
                        @focused_kyou="(...kyou: any[]) => emits('focused_kyou', kyou[0])"
                        @clicked_kyou="(...kyou: any[]) => { emits('focused_kyou', kyou[0]); emits('clicked_kyou', kyou[0]) }"
                        @deleted_kyou="(...kyou: any[]) => emits('deleted_kyou', kyou[0])"
                        @deleted_tag="(...tag: any[]) => emits('deleted_tag', tag[0])"
                        @deleted_text="(...text: any[]) => emits('deleted_text', text[0])"
                        @deleted_notification="(...n: any[]) => emits('deleted_notification', n[0])"
                        @registered_kyou="(...kyou: any[]) => emits('registered_kyou', kyou[0])"
                        @registered_tag="(...tag: any[]) => emits('registered_tag', tag[0])"
                        @registered_text="(...text: any[]) => emits('registered_text', text[0])"
                        @registered_notification="(...n: any[]) => emits('registered_notification', n[0])"
                        @updated_kyou="(...kyou: any[]) => emits('updated_kyou', kyou[0])"
                        @updated_tag="(...tag: any[]) => emits('updated_tag', tag[0])"
                        @updated_text="(...text: any[]) => emits('updated_text', text[0])"
                        @updated_notification="(...n: any[]) => emits('updated_notification', n[0])"
                        @requested_open_rykv_dialog="(...params: any[]) => emits('requested_open_rykv_dialog', params[0], params[1], params[2])"
                        ref="dnote_item_list_views" />
                </td>
            </tr>
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
    on_cell_dragover,
    on_cell_drop,

    // Exposed methods
    load_aggregated_value,
    reset,
} = useDnoteItemTableView({ props, emits, model_value })

defineExpose({ load_aggregated_value, reset })
</script>

<style scoped>
.dnote_item_table_td {
    vertical-align: top;
    min-width: 210px;
    padding: 4px;
}
</style>
