<template>
    <v-dialog v-model="is_show_dialog">
        <EndTimeIsPlaingView :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="[kyou.generate_info_identifer()]" :kyou="kyou" :last_added_tag="last_added_tag"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')" @requested_close_dialog="hide()"
            @requested_update_check_kyous="(kyou, is_checked) => emits('requested_update_check_kyous', kyou, is_checked)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import EndTimeIsPlaingView from '../views/end-time-is-plaing-view.vue'
import type { EndTimeIsPlaingDialogProps } from './end-time-is-plaing-dialog-props'
import type { KyouViewEmits } from '../views/kyou-view-emits'

defineProps<EndTimeIsPlaingDialogProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
