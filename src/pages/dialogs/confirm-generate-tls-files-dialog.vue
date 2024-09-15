<template>
    <ConfirmGenerateTLSFilesView />
</template>
<script lang="ts" setup>
import type { ConfirmGenerateTLSFilesDialogEmits } from './confirm-generate-tls-files-dialog-emits';
import type { ConfirmGenerateTLSFilesDialogProps } from './confirm-generate-tls-files-dialog-props';
import ConfirmGenerateTLSFilesView from '../views/confirm-generate-tls-files-view.vue';

const props = defineProps<ConfirmGenerateTLSFilesDialogProps>();
const emits = defineEmits<ConfirmGenerateTLSFilesDialogEmits>();
</script>