<template>
    <EditTagView />
    <KyouView />
</template>
<script setup lang="ts">
import type { EditTagDialogProps } from './edit-tag-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import EditTagView from '../views/edit-tag-view.vue';
import KyouView from '../views/kyou-view.vue';

const props = defineProps<EditTagDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
defineExpose({ show, hide })

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>
