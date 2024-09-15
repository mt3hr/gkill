<template>
    <EditFolderView />
</template>
<script lang="ts" setup>
import type { EditFolderDialogEmits } from './edit-folder-dialog-emits';
import type { EditFolderDialogProps } from './edit-folder-dialog-props';
import EditFolderView from '../views/edit-folder-view.vue';

const props = defineProps<EditFolderDialogProps>();
const emits = defineEmits<EditFolderDialogEmits>();
</script>
