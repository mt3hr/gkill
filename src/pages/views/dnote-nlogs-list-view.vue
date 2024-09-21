<template>
    <DnoteNlogView v-for="kyou, index in sorted_nlog_kyous" :nlog_kyou="kyou" :application_config="application_config"
        :gkill_api="gkill_api" :highlight_targets="new Array<Kyou>()" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')" />
</template>
<script setup lang="ts">
import { computed, type Ref, ref } from 'vue';
import DnoteNlogView from './dnote-nlog-view.vue';
import type { DnoteNlogsListViewEmits } from './dnote-nlogs-list-view-emits';
import type { DnoteNlogsListViewProps } from './dnote-nlogs-list-view-props';
import { Kyou } from '@/classes/datas/kyou';

const props = defineProps<DnoteNlogsListViewProps>();
const emits = defineEmits<DnoteNlogsListViewEmits>();
const calcrated_total_amount = computed(() => {
    let total_amount: Number = 0
    props.nlog_kyous.forEach(async (nlog_kyou) => {
        const errors = await nlog_kyou.load_typed_nlog()
        if (errors && errors.length !== 0) {
            emits('received_errors', errors)
            return
        }
        const nlog = nlog_kyou.typed_nlog
        total_amount = total_amount.valueOf() + nlog.amount.valueOf()
    })
    return total_amount
});
const sorted_nlog_kyous: Ref<Array<Kyou>> = ref(props.nlog_kyous.concat());
</script>
