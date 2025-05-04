<template>
    <div>
        <h2>{{ model_value?.title ? model_value.title : "" }}</h2>
        <v-virtual-scroll :items="aggregated_items" :height="'50vh'" :width="200 + 8" ref="list_view">
            <template v-slot:default="{ item }">
                <AggregatedListItem :application_config="application_config" :gkill_api="gkill_api"
                    :dnote_list_query="model_value!" @received_errors="(errors) => emits('received_errors', errors)"
                    @received_messages="(messages) => emits('received_messages', messages)" :aggregated_item="item"
                    @requested_delete_dnote_list_query="(id) => emits('requested_delete_dnote_list_query', id)"
                    @requested_update_dnote_list_query="(dnote_list_query) => emits('requested_update_dnote_list_query', dnote_list_query)" />
            </template>
        </v-virtual-scroll>
    </div>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type { Kyou } from '../../classes/datas/kyou';
import type AggregatedItem from '../../classes/dnote/aggregate-grouping-list-result-record';
import AggregatedListItem from './aggregated-list-item.vue';
import { DnoteListAggregator } from '../../classes/dnote/dnote-list-aggregator';
import type DnoteListViewProps from './dnote-list-view-props';
import type DnoteListQuery from './dnote-list-query';
import type DnoteListViewEmits from './dnote-list-view-emits';

defineProps<DnoteListViewProps>()
defineExpose({ load_aggregate_grouping_list })
const emits = defineEmits<DnoteListViewEmits>()
const model_value = defineModel<DnoteListQuery>()
const aggregated_items: Ref<Array<AggregatedItem>> = ref(new Array<AggregatedItem>())

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
</script>