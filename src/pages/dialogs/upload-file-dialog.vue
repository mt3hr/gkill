<template>
    <UploadFileView />
</template>

<script lang="ts" setup>
import type { UploadFileDialogEmits } from './upload-file-dialog-emits';
import type { UploadFileDialogProps } from './upload-file-dialog-props';

import UploadFileView from '../views/upload-file-view.vue';

const props = defineProps<UploadFileDialogProps>();
const emits = defineEmits<UploadFileDialogEmits>();

</script>
