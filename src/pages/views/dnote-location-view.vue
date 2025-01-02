<template>
    <KyouView v-for="kyou in kyous" :key="kyou.id" :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :is_image_view="false" :kyou="kyou" :last_added_tag="last_added_tag"
        :show_checkbox="false" :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
        :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_timeis_plaing_end_button="true"
        @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
        @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou: Kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @requested_update_check_kyous="(kyou: Array<Kyou>, is_checked: boolean) => emits('requested_update_check_kyous', kyou, is_checked)" />
</template>
<script setup lang="ts">
import type { DnoteLocationViewEmits } from './dnote-location-view-emits'
import type { DnoteLocationViewProps } from './dnote-location-view-props'
import { computed } from 'vue'
import { Kyou } from '@/classes/datas/kyou'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

const props = defineProps<DnoteLocationViewProps>()
const emits = defineEmits<DnoteLocationViewEmits>()
const kyous = computed(() => {
    return timeiss.value.concat(kmemos.value)
})
const timeiss = computed(() => {
    const timeis_kyous: Array<Kyou> = new Array<Kyou>()
    props.timeis_or_kmemo_kyou.forEach(async (kyou) => {
        if (kyou.data_type !== "timeis") {
            return
        }
        timeis_kyous.push(kyou)
    })
    return timeis_kyous
})
const kmemos = computed(() => {
    const kmemo_kyous: Array<Kyou> = new Array<Kyou>()
    props.timeis_or_kmemo_kyou.forEach(async (kyou) => {
        if (kyou.data_type !== "kmemo") {
            return
        }
        kmemo_kyous.push(kyou)
    })
    return kmemo_kyous
})
</script>
