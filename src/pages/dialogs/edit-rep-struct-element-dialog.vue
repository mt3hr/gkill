<template>
    <EditRepStructElementView />
</template>
<script lang="ts" setup>
import type { EditRepStructElementDialogEmits } from './edit-rep-struct-element-dialog-emits';
import type { EditRepStructElementDialogProps } from './edit-rep-struct-element-dialog-props';
import EditRepStructElementView from '../views/edit-rep-struct-element-view.vue';

const props = defineProps<EditRepStructElementDialogProps>();
const emits = defineEmits<EditRepStructElementDialogEmits>();
</script>