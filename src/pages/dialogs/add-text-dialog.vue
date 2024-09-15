<template>
    <AddTextView />
    <KyouView />
</template>
<script setup lang="ts">
import type { AddTextDialogProps } from './add-text-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import AddTextView from '../views/add-text-view.vue';
import KyouView from '../views/kyou-view.vue';

const props = defineProps<AddTextDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
defineExpose({ show, hide })

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>
<style scoped></style>