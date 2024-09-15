<template>
    <EditTimeIsView />
    <KyouView />
</template>
<script setup lang="ts">
import type { EditTimeIsDialogProps } from './edit-time-is-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import EditTimeIsView from '../views/edit-time-is-view.vue';
import KyouView from '../views/kyou-view.vue';

const props = defineProps<EditTimeIsDialogProps>();
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
