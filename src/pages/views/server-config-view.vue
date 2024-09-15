<template>
    <ConfirmGenerateTLSFilesDialog />
    <CreateAccountDialog />
    <ManageAccountDialog />
</template>
<script setup lang="ts">
import type { ServerConfig } from '@/classes/datas/config/server-config';
import { ref, type Ref } from 'vue';
import ConfirmGenerateTLSFilesDialog from '../dialogs/confirm-generate-tls-files-dialog.vue';
import CreateAccountDialog from '../dialogs/create-account-dialog.vue';
import ManageAccountDialog from '../dialogs/manage-account-dialog.vue';
import type { ServerConfigViewEmits } from './server-config-view-emits';
import type { ServerConfigViewProps } from './server-config-view-props';

const props = defineProps<ServerConfigViewProps>();
const emits = defineEmits<ServerConfigViewEmits>();

const cloned_server_config: Ref<ServerConfig> = ref(await props.server_config.clone());
const is_local_only_access: Ref<boolean> = ref(cloned_server_config.value.is_local_only_access);
const address: Ref<string> = ref(cloned_server_config.value.address);
const enable_tls: Ref<boolean> = ref(cloned_server_config.value.enable_tls);
const tls_cert_file: Ref<string> = ref(cloned_server_config.value.tls_cert_file);
const tls_key_file: Ref<string> = ref(cloned_server_config.value.tls_key_file);
const open_directory_command: Ref<string> = ref(cloned_server_config.value.open_directory_command);
const open_file_command: Ref<string> = ref(cloned_server_config.value.open_file_command)
const urlog_timeout: Ref<Number> = ref(cloned_server_config.value.urlog_timeout)
const urlog_useragent: Ref<string> = ref(cloned_server_config.value.urlog_useragent);
const upload_size_limit_month: Ref<Number> = ref(cloned_server_config.value.upload_size_limit_month);
const user_data_directory: Ref<string> = ref(cloned_server_config.value.user_data_directory);
</script>
