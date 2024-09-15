<template>
    <AddNewKFTLTemplateStructElementView />
</template>
<script lang="ts" setup>
import type { AddNewKFTLTemplateStructElementDialogEmits } from './add-new-kftl-template-struct-element-dialog-emits';
import type { AddNewKFTLTemplateStructElementDialogProps } from './add-new-kftl-template-struct-element-dialog-props';
import AddNewKFTLTemplateStructElementView from '../views/add-new-kftl-template-struct-element-view.vue';

const props = defineProps<AddNewKFTLTemplateStructElementDialogProps>();
const emits = defineEmits<AddNewKFTLTemplateStructElementDialogEmits>();
</script>
