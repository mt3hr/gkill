<template>
    <EditTextView />
    <KyouView />
</template>
<script setup lang="ts">
import type { EditTextDialogProps } from './edit-text-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import EditTextView from '../views/edit-text-view.vue';
import KyouView from '../views/kyou-view.vue';

const props = defineProps<EditTextDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
defineExpose({ show, hide })

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>
