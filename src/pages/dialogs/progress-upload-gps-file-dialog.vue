<template>
    <ProgressUploadGPSFileView />
</template>
<script lang="ts" setup>
import type { ProgressUploadGPSFileDialogEmits } from './progress-upload-gps-file-dialog-emits';
import type { ProgressUploadGPSFileDialogProps } from './progress-upload-gps-file-dialog-props';
import ProgressUploadGPSFileView from '../views/progress-upload-gps-file-view.vue';
const props = defineProps<ProgressUploadGPSFileDialogProps>();
const emits = defineEmits<ProgressUploadGPSFileDialogEmits>();
</script>
