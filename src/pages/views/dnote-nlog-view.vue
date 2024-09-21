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
import type { DnoteNlogViewEmits } from './dnote-nlog-view-emits'
import type { DnoteNlogViewProps } from './dnote-nlog-view-props'
import { type Ref, ref, computed } from 'vue'
import { Kyou } from '@/classes/datas/kyou'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

const props = defineProps<DnoteNlogViewProps>()
const emits = defineEmits<DnoteNlogViewEmits>()
const cloned_kyou: Ref<Kyou> = ref(await props.nlog_kyou.clone())
const cloned_nlog = computed(async () => {
    const errors = await cloned_kyou.value.load_typed_nlog()
    if (errors && errors.length !== 0) {
        emits('received_errors', errors)
        return
    }
    const nlog = cloned_kyou.value.typed_nlog
    return nlog
})
</script>
