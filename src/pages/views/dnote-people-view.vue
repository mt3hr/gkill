<template>
    <KyouView :application_config="application_config" :gkill_api="gkill_api" :highlight_targets="highlight_targets"
        :is_image_view="false" :kyou="timeis_or_kmemo_kyou" :last_added_tag="last_added_tag" :show_checkbox="false"
        :show_content_only="false" :show_mi_create_time="true" :show_mi_estimate_end_time="true"
        :show_mi_estimate_start_time="true" :show_mi_limit_time="true" :show_timeis_plaing_end_button="true"
        @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
        @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou: Kyou) => emits('requested_reload_kyou', kyou)"
        @requested_reload_list="emits('requested_reload_list')"
        @requested_update_check_kyous="(kyou: Array<Kyou>, is_checked: boolean) => emits('requested_update_check_kyous', kyou, is_checked)" />
</template>
<script setup lang="ts">
import type { DnotePeopleViewEmits } from './dnote-people-view-emits'
import type { DnotePeopleViewProps } from './dnote-people-view-props'
import KyouDialog from '../dialogs/kyou-dialog.vue'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { GkillError } from '@/classes/api/gkill-error'
import type { Kyou } from '@/classes/datas/kyou'
import type { Kmemo } from '@/classes/datas/kmemo'
import type { TimeIs } from '@/classes/datas/time-is'
import { type Ref, ref, computed } from 'vue'

const props = defineProps<DnotePeopleViewProps>()
const emits = defineEmits<DnotePeopleViewEmits>()
</script>
