<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmDeleteNotificationView :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="[notification.generate_info_identifer()]" :kyou="kyou" :last_added_tag="last_added_tag"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :notification="notification"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')" @requested_close_dialog="hide()" />
    </v-dialog>
</template>
<script setup lang="ts">
import ConfirmDeleteNotificationView from '../views/confirm-delete-notification-view.vue';
import { type Ref, ref } from 'vue'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import type { ConfirmDeleteNotificationDialogProps } from './confirm-delete-notification-dialog-props';

defineProps<ConfirmDeleteNotificationDialogProps>()
const emits = defineEmits<KyouDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
