<template>
    <div>
        <span class="title">
            <span>{{ title }}</span>
            <span>:</span>
        </span>
        <span class="value">
            <span>{{ prefix }}</span>
            <span>{{ value }}</span>
            <span>{{ suffix }}</span>
        </span>
        <!-- //TODO matchKyousのダイアログ -->
    </div>
</template>
<script lang="ts" setup>
import { ref, type Ref } from 'vue'
import type DnoteItemProps from './dnote-item-props'
import type { Kyou } from '../../classes/datas/kyou';
import { DnoteAggregator } from '../../classes/dnote/dnote-aggregator';
import type { FindKyouQuery } from '../../classes/api/find_query/find-kyou-query';

const props = defineProps<DnoteItemProps>()
defineExpose({ load_aggregated_value })

const title: Ref<string> = ref(props.dnote_item.title)
const prefix: Ref<string> = ref(props.dnote_item.prefix)
const suffix: Ref<string> = ref(props.dnote_item.suffix)
const value: Ref<string> = ref("")
const related_kyous: Ref<Array<Kyou>> = ref(new Array<Kyou>())

async function load_aggregated_value(abort_controller: AbortController, kyous: Array<Kyou>, query: FindKyouQuery, kyou_is_loaded: boolean) {
    const dnote_aggregator = new DnoteAggregator(props.dnote_item.predicate, props.dnote_item.aggregate_target)
    const aggregate_result = await dnote_aggregator.aggregate(abort_controller, kyous, query, kyou_is_loaded)
    value.value = aggregate_result.result_string
    related_kyous.value = aggregate_result.match_kyous
}

</script>
<style lang="css">
.title {
    font-weight: bold;
}

.value {
    text-align: right;
}
</style>