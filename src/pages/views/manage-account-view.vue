<template>
    <AllocateRepDialog :application_config="application_config" :gkill_api="gkill_api" :server_config="server_config"
        :user_id="focused_user_id"
        @requested_reload_server_config="(server_config) => emits('requested_reload_server_config', server_config)"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    <ConfirmGenerateTLSFilesDialog :application_config="application_config" :gkill_api="gkill_api"
        :server_config="server_config" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    <ConfirmResetPasswordDialog :application_config="application_config" :gkill_api="gkill_api"
        :server_config="server_config" :user_id="focused_user_id"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    <CreateAccountDialog :application_config="application_config" :gkill_api="gkill_api" :server_config="server_config"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    <ShowPasswordResetLinkDialog :application_config="application_config" :gkill_api="gkill_api"
        :server_config="server_config" :user_id="focused_user_id"
        @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import type { ManageAccountViewEmits } from './manage-account-view-emits'
import type { ManageAccountViewProps } from './manage-account-view-props'
import { ServerConfig } from '@/classes/datas/config/server-config'
import AllocateRepDialog from '../dialogs/allocate-rep-dialog.vue'
import ConfirmGenerateTLSFilesDialog from '../dialogs/confirm-generate-tls-files-dialog.vue'
import ConfirmResetPasswordDialog from '../dialogs/confirm-reset-password-dialog.vue'
import CreateAccountDialog from '../dialogs/create-account-dialog.vue'
import ShowPasswordResetLinkDialog from '../dialogs/show-password-reset-link-dialog.vue'

const props = defineProps<ManageAccountViewProps>()
const emits = defineEmits<ManageAccountViewEmits>()
const cloned_server_config: Ref<ServerConfig> = ref(await props.server_config.clone())
const focused_user_id: Ref<string> = ref("")
</script>
