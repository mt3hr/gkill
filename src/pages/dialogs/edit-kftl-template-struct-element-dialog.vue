<template>
    <EditKFTLTemplateStructElementView />
</template>
<script lang="ts" setup>
import type { EditKFTLTemplateStructElementDialogEmits } from './edit-kftl-template-struct-element-dialog-emits';
import type { EditKFTLTemplateStructElementDialogProps } from './edit-kftl-template-struct-element-dialog-props';
import EditKFTLTemplateStructElementView from '../views/edit-kftl-template-struct-element-view.vue';

const props = defineProps<EditKFTLTemplateStructElementDialogProps>();
const emits = defineEmits<EditKFTLTemplateStructElementDialogEmits>();
</script>
