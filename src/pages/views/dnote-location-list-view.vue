<template>
    <DnoteLocationView :timeis_or_kmemo_kyou="timeis_kyous" :application_config="application_config"
        :gkill_api="gkill_api" :last_added_tag="last_added_tag" :highlight_targets="new Array<Kyou>()"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')" />
</template>
<script lang="ts" setup>
import type { DnoteLocationListViewEmits } from './dnote-location-list-view-emits'
import type { DnoteLocationListViewProps } from './dnote-location-list-view-props'
import type { Kyou } from '@/classes/datas/kyou'
import DnoteLocationView from './dnote-location-view.vue'

defineProps<DnoteLocationListViewProps>()
const emits = defineEmits<DnoteLocationListViewEmits>()
</script>
