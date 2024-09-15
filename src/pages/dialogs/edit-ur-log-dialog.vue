<template>
    <EditURLogView />
    <KyouView />
</template>
<script setup lang="ts">
import type { EditURLogDialogProps } from './edit-ur-log-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import EditURLogView from '../views/edit-ur-log-view.vue';
import KyouView from '../views/kyou-view.vue';

const props = defineProps<EditURLogDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
defineExpose({ show, hide })

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>
