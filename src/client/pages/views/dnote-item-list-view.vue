<template>
    <div class="dnote_item_list_root" @dragover="on_list_dragover" @drop="on_list_drop">
        <DnoteItemView v-for="(dnote_item, index) in model_value" :key="dnote_item.id" v-model="model_value[index]"
            :editable="editable" :dnd_list_index="dnd_list_index" :application_config="application_config"
            :gkill_api="gkill_api"
            @requested_move_dnote_item="(...args: any[]) => emits('requested_move_dnote_item', args[0] as string, args[1] as number, args[2] as string, args[3] as number, args[4] as 'up' | 'down')"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0])"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0])"
            @focused_kyou="(...kyou: any[]) => emits('focused_kyou', kyou[0])"
            @clicked_kyou="(...kyou: any[]) => { emits('focused_kyou', kyou[0]); emits('clicked_kyou', kyou[0]) }"
            @requested_delete_dnote_item="(...id: any[]) => delete_dnote_item(id[0] as string)"
            @requested_update_dnote_item="(...d: any[]) => update_dnote_item(d[0])"
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
            @finish_a_aggregate_task="emits('finish_a_aggregate_task')" ref="dnote_item_views" />
    </div>
</template>

<script lang="ts" setup>
import DnoteItemView from "./dnote-item-view.vue"
import type DnoteItemListViewProps from "./dnote-item-list-view-props"
import type DnoteItemListViewEmits from "./dnote-item-list-view-emits"
import type DnoteItem from "../../classes/dnote/dnote-item"
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
