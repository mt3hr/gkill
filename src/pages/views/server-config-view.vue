<template>
    <ConfirmGenerateTLSFilesDialog :application_config="application_config" :gkill_api="gkill_api"
        :server_config="props.server_config" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    <CreateAccountDialog :application_config="application_config" :gkill_api="gkill_api"
        :server_config="props.server_config" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
    <ManageAccountDialog :application_config="application_config" :gkill_api="gkill_api"
        :server_config="props.server_config" @received_errors="(errors) => emits('received_errors', errors)"
        @received_messages="(messages) => emits('received_messages', messages)" />
</template>
<script setup lang="ts">
import type { ServerConfig } from '@/classes/datas/config/server-config'
import { computed, ref, type Ref } from 'vue'
import ConfirmGenerateTLSFilesDialog from '../dialogs/confirm-generate-tls-files-dialog.vue'
import CreateAccountDialog from '../dialogs/create-account-dialog.vue'
import ManageAccountDialog from '../dialogs/manage-account-dialog.vue'
import type { ServerConfigViewEmits } from './server-config-view-emits'
import type { ServerConfigViewProps } from './server-config-view-props'

const props = defineProps<ServerConfigViewProps>()
const emits = defineEmits<ServerConfigViewEmits>()

const is_local_only_access = computed(() => (props.server_config.is_local_only_access))
const address = computed(() => (props.server_config.address))
const enable_tls = computed(() => (props.server_config.enable_tls))
const tls_cert_file = computed(() => (props.server_config.tls_cert_file))
const tls_key_file = computed(() => (props.server_config.tls_key_file))
const open_directory_command = computed(() => (props.server_config.open_directory_command))
const open_file_command = computed(() => (props.server_config.open_file_command))
const urlog_timeout = computed(() => (props.server_config.urlog_timeout))
const urlog_useragent = computed(() => (props.server_config.urlog_useragent))
const upload_size_limit_month = computed(() => (props.server_config.upload_size_limit_month))
const user_data_directory = computed(() => (props.server_config.user_data_directory))
</script>
