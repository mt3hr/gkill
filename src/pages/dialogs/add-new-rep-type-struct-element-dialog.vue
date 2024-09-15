<template>
    <AddNewRepTypeStructElementView />
</template>
<script lang="ts" setup>
import type { AddNewRepTypeStructElementDialogEmits } from './add-new-rep-type-struct-element-dialog-emits';
import type { AddNewRepTypeStructElementDialogProps } from './add-new-rep-type-struct-element-dialog-props';
import AddNewRepTypeStructElementView from '../views/add-new-rep-type-struct-element-view.vue';

const props = defineProps<AddNewRepTypeStructElementDialogProps>();
const emits = defineEmits<AddNewRepTypeStructElementDialogEmits>();
</script>