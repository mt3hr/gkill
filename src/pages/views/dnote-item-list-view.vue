<template>
    <div>
        <DnoteItemView v-for="dnote_item in model_value" :dnote_item="dnote_item" :key="dnote_item.id"
            :editable="editable" :application_config="application_config" :gkill_api="gkill_api"
            ref="dnote_item_views" />
    </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue';
import type DnoteItemListViewProps from './dnote-item-list-view-props';
import DnoteItemView from './dnote-item-view.vue';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';
import type { Kyou } from '../../classes/datas/kyou';
import type DnoteItem from '../../classes/dnote/dnote-item';

const dnote_item_views = ref()

defineProps<DnoteItemListViewProps>()
defineExpose({ load_aggregated_value })
const model_value = defineModel<Array<DnoteItem>>()

async function load_aggregated_value(abort_controller: AbortController, kyous: Array<Kyou>, query: FindKyouQuery, kyou_is_loaded: boolean) {
    if (!dnote_item_views.value || !model_value.value) {
        return
    }
    for (let i = 0; i < dnote_item_views.value.length; i++) {
        if (!(dnote_item_views.value as any)[i]) {
            continue
        }
        const dnote_item_view = (dnote_item_views.value as any)[i]
        await dnote_item_view.load_aggregated_value(abort_controller, kyous, query, kyou_is_loaded)
    }
}

</script>