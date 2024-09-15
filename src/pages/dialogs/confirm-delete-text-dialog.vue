<template>
    <ConfirmDeleteTextView />
    <KyouView />
</template>
<script setup lang="ts">
import type { ConfirmDeleteTextDialogProps } from './confirm-delete-text-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import ConfirmDeleteTextView from '../views/confirm-delete-text-view.vue';
import KyouView from '../views/kyou-view.vue';

const props = defineProps<ConfirmDeleteTextDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
defineExpose({ show, hide })

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>