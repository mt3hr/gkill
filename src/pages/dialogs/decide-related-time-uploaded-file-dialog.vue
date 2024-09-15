<template>
    <DecideRelatedTimeUploadedFileView />
</template>
<script lang="ts" setup>
// ˅
import type { DecideRelatedTimeUploadedFileDialogEmits } from './decide-related-time-uploaded-file-dialog-emits';
import type { DecideRelatedTimeUploadedFileDialogProps } from './decide-related-time-uploaded-file-dialog-props';
import DecideRelatedTimeUploadedFileView from '../views/decide-related-time-uploaded-file-view.vue';

const props = defineProps<DecideRelatedTimeUploadedFileDialogProps>();
const emits = defineEmits<DecideRelatedTimeUploadedFileDialogEmits>();

</script>