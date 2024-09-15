<template>
    <AddNewTagStructElementView />
</template>
<script lang="ts" setup>
import type { AddNewTagStructElementDialogEmits } from './add-new-tag-struct-element-dialog-emits';
import type { AddNewTagStructElementDialogProps } from './add-new-tag-struct-element-dialog-props';
import AddNewTagStructElementView from '../views/add-new-tag-struct-element-view.vue';

const props = defineProps<AddNewTagStructElementDialogProps>();
const emits = defineEmits<AddNewTagStructElementDialogEmits>();
</script>