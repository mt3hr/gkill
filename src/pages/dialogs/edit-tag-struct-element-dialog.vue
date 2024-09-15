<template>
    <EditTagStructElementView />
</template>
<script lang="ts" setup>
import type { EditTagStructElementDialogEmits } from './edit-tag-struct-element-dialog-emits';
import type { EditTagStructElementDialogProps } from './edit-tag-struct-element-dialog-props';
import EditTagStructElementView from '../views/edit-tag-struct-element-view.vue';

const props = defineProps<EditTagStructElementDialogProps>();
const emits = defineEmits<EditTagStructElementDialogEmits>();
</script>
