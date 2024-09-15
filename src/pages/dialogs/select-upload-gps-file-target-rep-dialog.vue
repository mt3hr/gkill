<template>
    <SelectUploadGPSFileTargetRepView />
</template>
<script lang="ts" setup>
import type { SelectUploadGPSFileTargetRepDialogEmits } from './select-upload-gps-file-target-rep-dialog-emits';
import type { SelectUploadGPSFileTargetRepDialogProps } from './select-upload-gps-file-target-rep-dialog-props';
import SelectUploadGPSFileTargetRepView from '../views/select-upload-gps-file-target-rep-view.vue';

const props = defineProps<SelectUploadGPSFileTargetRepDialogProps>();
const emits = defineEmits<SelectUploadGPSFileTargetRepDialogEmits>();
</script>
