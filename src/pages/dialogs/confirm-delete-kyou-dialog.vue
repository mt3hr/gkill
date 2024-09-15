<template>
    <ConfirmDeleteKyouView />
    <KyouView />
</template>
<script setup lang="ts">
import type { ConfirmDeleteKyouDialogProps } from './confirm-delete-kyou-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import ConfirmDeleteKyouView from '../views/confirm-delete-kyou-view.vue';
import KyouView from '../views/kyou-view.vue';

const props = defineProps<ConfirmDeleteKyouDialogProps>();
const emits = defineEmits<KyouDialogEmits>();

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>
