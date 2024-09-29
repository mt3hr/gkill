<template>
    <NlogContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script setup lang="ts">
import type { NlogViewProps } from './nlog-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import type { Nlog } from '@/classes/datas/nlog'
import { type Ref, ref } from 'vue'
import NlogContextMenu from './nlog-context-menu.vue'
import type { Kyou } from '@/classes/datas/kyou'

const props = defineProps<NlogViewProps>()
const emits = defineEmits<KyouViewEmits>()
</script>
