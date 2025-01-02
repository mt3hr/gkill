<template>
    <DnoteNlogView v-for="kyou in sorted_nlog_kyous" :key="kyou.id" :nlog_kyou="kyou"
        :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="new Array<Kyou>()"
        :last_added_tag="last_added_tag" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')" />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import DnoteNlogView from './dnote-nlog-view.vue'
import type { DnoteNlogsListViewEmits } from './dnote-nlogs-list-view-emits'
import type { DnoteNlogsListViewProps } from './dnote-nlogs-list-view-props'
import { Kyou } from '@/classes/datas/kyou'

const props = defineProps<DnoteNlogsListViewProps>()
const emits = defineEmits<DnoteNlogsListViewEmits>()
const sorted_nlog_kyous: Ref<Array<Kyou>> = ref(props.nlog_kyous.concat())
</script>
