<template>
    <AllocateRepDialog />
    <ConfirmGenerateTLSFilesDialog />
    <ConfirmResetPasswordDialog />
    <CreateAccountDialog />
    <ShowPasswordResetLinkDialog />
</template>
<script setup lang="ts">
/* __placeholder__ */
import { type Ref, ref } from 'vue';
import type { ManageAccountViewEmits } from './manage-account-view-emits';
import type { ManageAccountViewProps } from './manage-account-view-props';
import { ServerConfig } from '@/classes/datas/config/server-config';
import AllocateRepDialog from '../dialogs/allocate-rep-dialog.vue';
import ConfirmGenerateTLSFilesDialog from '../dialogs/confirm-generate-tls-files-dialog.vue';
import ConfirmResetPasswordDialog from '../dialogs/confirm-reset-password-dialog.vue';
import CreateAccountDialog from '../dialogs/create-account-dialog.vue';
import ShowPasswordResetLinkDialog from '../dialogs/show-password-reset-link-dialog.vue';

const props = defineProps<ManageAccountViewProps>();
const emits = defineEmits<ManageAccountViewEmits>();
const cloned_server_config: Ref<ServerConfig> = ref(await props.server_config.clone());
</script>
