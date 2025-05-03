<template>
    <div>
        <table>
            <tr>
                <td v-for="dnote_list_item in model_value" :key="dnote_list_item.id">
                    <DnoteListView :application_config="application_config" :gkill_api="gkill_api"
                        v-model="dnote_list_item!" ref="dnote_list_views" />
                </td>
            </tr>
        </table>
    </div>
</template>
<script lang="ts" setup>
import { ref } from 'vue';
import type DnoteListTableViewProps from './dnote-list-table-view-props';
import DnoteListView from './dnote-list-view.vue';
import type { Kyou } from '../../classes/datas/kyou';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type DnoteListQuery from './dnote-list-query';

const dnote_list_views = ref()

defineProps<DnoteListTableViewProps>()
defineExpose({ load_aggregate_grouping_list })
const model_value = defineModel<Array<DnoteListQuery>>()

async function load_aggregate_grouping_list(abort_controller: AbortController, kyous: Array<Kyou>, find_kyou_query: FindKyouQuery, kyou_is_loaded: boolean): Promise<void> {
    if (!dnote_list_views.value || !model_value.value) {
        return
    }
    for (let i = 0; i < dnote_list_views.value.length; i++) {
        if (!dnote_list_views.value[i]) {
            continue
        }
        await dnote_list_views.value[i].load_aggregate_grouping_list(abort_controller, kyous, find_kyou_query, kyou_is_loaded)
    }
}
</script>