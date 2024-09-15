<template>
    <AddNewRepStructElementView />
</template>
<script lang="ts" setup>
import type { AddNewRepStructElementDialogEmits } from './add-new-rep-struct-element-dialog-emits';
import type { AddNewRepStructElementDialogProps } from './add-new-rep-struct-element-dialog-props';
import AddNewRepStructElementView from '../views/add-new-rep-struct-element-view.vue';

const props = defineProps<AddNewRepStructElementDialogProps>();
const emits = defineEmits<AddNewRepStructElementDialogEmits>();
</script>
