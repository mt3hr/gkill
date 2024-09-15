<template>
    <SelectUploadFileTargetRepView />
</template>
<script lang="ts" setup>
import type { SelectUploadFileTargetRepDialogEmits } from './select-upload-file-target-rep-dialog-emits';
import type { SelectUploadFileTargetRepDialogProps } from './select-upload-file-target-rep-dialog-props';
import SelectUploadFileTargetRepView from '../views/select-upload-file-target-rep-view.vue';

const props = defineProps<SelectUploadFileTargetRepDialogProps>();
const emits = defineEmits<SelectUploadFileTargetRepDialogEmits>();
</script>
