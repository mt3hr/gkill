<template>
    <ReKyouContextMenu :application_config="application_config" :gkill_api="gkill_api"
        :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)"
        @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)" @requested_reload_list="() => { }"
        @requested_update_check_kyous="(kyous, is_checked) => emits('requested_update_check_kyous', kyous, is_checked)" />
</template>
<script setup lang="ts">
import type { ReKyouViewProps } from './re-kyou-view-props'
import type { KyouViewEmits } from './kyou-view-emits'
import ReKyouContextMenu from './re-kyou-context-menu.vue'
import { computed, type Ref, ref } from 'vue'
import { GetKyouRequest } from '@/classes/api/req_res/get-kyou-request'
import { ReKyou } from '@/classes/datas/re-kyou'
import type { Kyou } from '@/classes/datas/kyou'

const props = defineProps<ReKyouViewProps>()
const emits = defineEmits<KyouViewEmits>()

const target_kyou = computed(async () => {
    const req = new GetKyouRequest()
    req.id = props.rekyou.target_id
    const res = await props.gkill_api.get_kyou(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    return res.kyou_histories[0]
})
</script>
