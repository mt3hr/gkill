<template>
    <div>
        <h2>{{ model_value?.title ? model_value.title : "" }}</h2>
        <v-virtual-scroll :items="aggregated_items" :height="'50vh'" :width="200 + 8"
            @contextmenu.prevent.stop="(e: any) => { if (editable) { contextmenu?.show(e, model_value!.id) } }"
            ref=" list_view">
            <template v-slot:default="{ item }">
                <AggregatedListItem :application_config="application_config" :gkill_api="gkill_api"
                    :dnote_list_query="model_value!" @received_errors="(errors) => emits('received_errors', errors)"
                    @received_messages="(messages) => emits('received_messages', messages)" :aggregated_item="item"
                    @requested_delete_dnote_list_query="(id) => emits('requested_delete_dnote_list_query', id)"
                    @requested_update_dnote_list_query="(dnote_list_query) => emits('requested_update_dnote_list_query', dnote_list_query)"
                    @deleted_kyou="(kyou) => emits('deleted_kyou', kyou)"
                    @deleted_tag="(tag) => emits('deleted_tag', tag)"
                    @deleted_text="(text) => emits('deleted_text', text)"
                    @deleted_notification="(notification) => emits('deleted_notification', notification)"
                    @registered_kyou="(kyou) => emits('registered_kyou', kyou)"
                    @registered_tag="(tag) => emits('registered_tag', tag)"
                    @registered_text="(text) => emits('registered_text', text)"
                    @registered_notification="(notification) => emits('registered_notification', notification)"
                    @updated_kyou="(kyou) => emits('updated_kyou', kyou)"
                    @updated_tag="(tag) => emits('updated_tag', tag)"
                    @updated_text="(text) => emits('updated_text', text)"
                    @updated_notification="(notification) => emits('updated_notification', notification)" />
            </template>
        </v-virtual-scroll>
        <DnoteListQueryContextMenu :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_delete_dnote_list_query="confirm_delete_dnote_list_query_dialog?.show(model_value!)"
            @requested_edit_dnote_list_query="edit_dnote_list_query?.show()" ref="contextmenu" />
        <ConfirmDeleteDnoteListQueryDialog :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_delete_dnote_list_query="(id) => emits('requested_delete_dnote_list_query', id)"
            ref="confirm_delete_dnote_list_query_dialog" />
        <EditDnoteListDialog :application_config="application_config" :gkill_api="gkill_api"
            :dnote_list_query="model_value!" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_update_dnote_list_query="(dnote_list_query) => emits('requested_update_dnote_list_query', dnote_list_query)"
            ref="edit_dnote_list_query" />
    </div>
</template>
<script lang="ts" setup>
import { nextTick, type Ref, ref } from 'vue';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type { Kyou } from '../../classes/datas/kyou';
import type AgregatedItem from '../../classes/dnote/aggregate-grouping-list-result-record';
import AggregatedListItem from './aggregated-list-item.vue';
import { DnoteListAggregator } from '../../classes/dnote/dnote-list-aggregator';
import type DnoteListViewProps from './dnote-list-view-props';
import type DnoteListQuery from './dnote-list-query';
import type DnoteListViewEmits from './dnote-list-view-emits';
import EditDnoteListDialog from '../dialogs/edit-dnote-list-dialog.vue';
import DnoteListQueryContextMenu from './dnote-list-query-context-menu.vue';
import ConfirmDeleteDnoteListQueryDialog from '../dialogs/confirm-delete-dnote-list-query-dialog.vue';

const contextmenu = ref<InstanceType<typeof DnoteListQueryContextMenu> | null>(null);
const confirm_delete_dnote_list_query_dialog = ref<InstanceType<typeof ConfirmDeleteDnoteListQueryDialog> | null>(null);
const edit_dnote_list_query = ref<InstanceType<typeof EditDnoteListDialog> | null>(null);


defineProps<DnoteListViewProps>()
defineExpose({ load_aggregate_grouping_list, reset })
const emits = defineEmits<DnoteListViewEmits>()
const model_value = defineModel<DnoteListQuery>()
const aggregated_items: Ref<Array<AgregatedItem>> = ref(new Array<AgregatedItem>())

async function load_aggregate_grouping_list(abort_controller: AbortController, kyous: Array<Kyou>, find_kyou_query: FindKyouQuery, kyou_is_loaded: boolean): Promise<void> {
    if (!model_value.value) {
        return
    }
    const list_aggregator = new DnoteListAggregator(model_value.value.predicate, model_value.value.key_getter, model_value.value.aggregate_target)
    const aggregated_result = await list_aggregator.aggregate_grouping_list(abort_controller, kyous, find_kyou_query, kyou_is_loaded)

    aggregated_items.value.splice(0)
    for (let i = 0; i < aggregated_result.length; i++) {
        aggregated_items.value.push(aggregated_result[i])
    }
}

async function reset(): Promise<void> {
    return nextTick(async () => {
        aggregated_items.value.splice(0)
    })
}
</script>