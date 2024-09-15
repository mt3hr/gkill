<template>
    <EditRepTypeStructElementView />
</template>
<script lang="ts" setup>
import type { EditRepTypeStructElementDialogEmits } from './edit-rep-type-struct-element-dialog-emits';
import type { EditRepTypeStructElementDialogProps } from './edit-rep-type-struct-element-dialog-props';
import EditRepTypeStructElementView from '../views/edit-rep-type-struct-element-view.vue';

const props = defineProps<EditRepTypeStructElementDialogProps>();
const emits = defineEmits<EditRepTypeStructElementDialogEmits>();
</script>
