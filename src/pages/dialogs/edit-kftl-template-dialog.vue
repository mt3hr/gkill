<template>
    <EditKFTLTemplateView />
</template>
<script lang="ts" setup>
import type { EditKFTLTemplateDialogEmits } from './edit-kftl-template-dialog-emits';
import type { EditKFTLTemplateDialogProps } from './edit-kftl-template-dialog-props';
import EditKFTLTemplateView from '../views/edit-kftl-template-view.vue';
const props = defineProps<EditKFTLTemplateDialogProps>();
const emits = defineEmits<EditKFTLTemplateDialogEmits>();
</script>
