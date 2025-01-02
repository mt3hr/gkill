<template>
    <DnotePeopleView v-for="kyou in etxructed_sorted_people_kyous" :key="kyou.id"
        :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="new Array<Kyou>()"
        :last_added_tag="last_added_tag" :timeis_or_kmemo_kyou="kyou"
        @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
        @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou: Kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @requested_update_check_kyous="(kyou: Array<Kyou>, is_checked: boolean) => emits('requested_update_check_kyous', kyou, is_checked)" />
</template>
<script setup lang="ts">
import type { DnotePeoplesListViewEmits } from './dnote-peoples-list-view-emits'
import type { DnotePeoplesListViewProps } from './dnote-peoples-list-view-props'
import type { Kyou } from '@/classes/datas/kyou'
import { type Ref, ref } from 'vue'
import DnotePeopleView from './dnote-people-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

const props = defineProps<DnotePeoplesListViewProps>()
const emits = defineEmits<DnotePeoplesListViewEmits>()
const etxructed_sorted_people_kyous: Ref<Array<Kyou>> = ref(props.timeis_kyous.concat())
</script>
