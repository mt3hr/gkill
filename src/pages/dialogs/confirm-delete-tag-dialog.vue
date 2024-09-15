<template>
    <KyouView />
    <ConfirmDeleteTagView />
</template>
<script setup lang="ts">
import type { ConfirmDeleteTagDialogProps } from './confirm-delete-tag-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import ConfirmDeleteTagView from '../views/confirm-delete-tag-view.vue';
import KyouView from '../views/kyou-view.vue';

const props = defineProps<ConfirmDeleteTagDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
defineExpose({ show, hide })

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>
