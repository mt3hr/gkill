<template>
    <ProgressUploadFileView />
</template>
<script lang="ts" setup>
import type { ProgressUploadFileDialogEmits } from './progress-upload-file-dialog-emits';
import type { ProgressUploadFileDialogProps } from './progress-upload-file-dialog-props';
import ProgressUploadFileView from '../views/progress-upload-file-view.vue';

const props = defineProps<ProgressUploadFileDialogProps>();
const emits = defineEmits<ProgressUploadFileDialogEmits>();
</script>
