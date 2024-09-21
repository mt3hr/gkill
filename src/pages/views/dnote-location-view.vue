<template>
    <KyouView :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
        :is_image_view="false" :kyou="cloned_kyou" :last_added_tag="last_added_tag" :show_checkbox="false"
        :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
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
import { type Ref, ref, computed } from 'vue'
import { Kyou } from '@/classes/datas/kyou'
import type { TimeIs } from '@/classes/datas/time-is'
import type { GkillError } from '@/classes/api/gkill-error'
import type { Kmemo } from '@/classes/datas/kmemo'
import type { GkillMessage } from '@/classes/api/gkill-message'

const props = defineProps<DnoteLocationViewProps>()
const emits = defineEmits<DnoteLocationViewEmits>()
const cloned_kyou: Ref<Array<Kyou>> = ref(props.timeis_or_kmemo_kyou.concat())
const cloned_timeis = computed(() => {
    const timeiss: Array<TimeIs> = new Array<TimeIs>()
    cloned_kyou.value.forEach(async (kyou) => {
        if (kyou.data_type !== "timeis") {
            return
        }
        const errors: Array<GkillError> = await kyou.load_typed_timeis()
        if (errors && errors.length !== 0) {
            emits('received_errors', errors)
            return
        }
        timeiss.push(kyou.typed_timeis)
    })
    return timeiss
})
const cloned_kmemo = computed(() => {
    const kmemos: Array<Kmemo> = new Array<Kmemo>()
    cloned_kyou.value.forEach(async (kyou) => {
        if (kyou.data_type !== "kmemo") {
            return
        }
        const errors: Array<GkillError> = await kyou.load_typed_kmemo()
        if (errors && errors.length !== 0) {
            emits('received_errors', errors)
            return
        }
        kmemos.push(kyou.typed_kmemo)
    })
    return kmemos
})
</script>
