<template>
    <div class="dnote_item_table_root">
        <table class="dnote_item_table">
            <tr>
                <td v-for="(list, listIndex) in model_value" :key="listIndex" class="dnote_item_table_td"
                    @dragover="on_cell_dragover" @drop="(e) => on_cell_drop(e, listIndex)">
                    <DnoteItemListView v-model="model_value[listIndex]" :dnd_list_index="listIndex" :editable="editable"
                        :application_config="application_config" :gkill_api="gkill_api"
                        @requested_move_dnote_item="(list_id: string, list_index: number, item_id: string, item_index: number, direction: 'up' | 'down') => handle_move_dnote_item(list_id, list_index, item_id, item_index, direction)"
                        @finish_a_aggregate_task="emits('finish_a_aggregate_task')"
                        @focused_kyou="(kyou: Kyou) => emits('focused_kyou', kyou)"
                        @clicked_kyou="(kyou: Kyou) => { emits('focused_kyou', kyou); emits('clicked_kyou', kyou) }"
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
                        ref="dnote_item_list_views" />
                </td>
            </tr>
        </table>
    </div>
</template>

<script lang="ts" setup>
import DnoteItemListView from "./dnote-item-list-view.vue"
import type { RykvDialogKind, RykvDialogPayload } from "./rykv-dialog-kind"
import type { Kyou } from "@/classes/datas/kyou"
import type { Tag } from "@/classes/datas/tag"
import type { Text } from "@/classes/datas/text"
import type { Notification } from "@/classes/datas/notification"
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
    padding: 0px;
}
</style>
