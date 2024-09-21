<template>
    <ShowPasswordResetLinkDialog :application_config="application_config" :gkill_api="gkill_api"
        :server_config="server_config" :user_id="user_id"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
</template>
<script setup lang="ts">
import type { ConfirmResetPasswordViewEmits } from './confirm-reset-password-view-emits'
import type { ConfirmResetPasswordViewProps } from './confirm-reset-password-view-props'
import ShowPasswordResetLinkDialog from '../dialogs/show-password-reset-link-dialog.vue'

const props = defineProps<ConfirmResetPasswordViewProps>()
const emits = defineEmits<ConfirmResetPasswordViewEmits>()
</script>
