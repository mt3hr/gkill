<template>
    <EditNlogView />
    <KyouView />
</template>
<script setup lang="ts">
import type { EditNlogDialogProps } from './edit-nlog-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import EditNlogView from '../views/edit-nlog-view.vue';
import KyouView from '../views/kyou-view.vue';

const props = defineProps<EditNlogDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
defineExpose({ show, hide })

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>