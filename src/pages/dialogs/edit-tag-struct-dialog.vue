<template>
    <EditTagStructView />
</template>
<script lang="ts" setup>
import type { EditTagStructDialogEmits } from './edit-tag-struct-dialog-emits';
import type { EditTagStructDialogProps } from './edit-tag-struct-dialog-props';
import EditTagStructView from '../views/edit-tag-struct-view.vue';

const props = defineProps<EditTagStructDialogProps>();
const emits = defineEmits<EditTagStructDialogEmits>();
</script>
