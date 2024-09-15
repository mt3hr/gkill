<template>
    <EditIDFKyouView />
    <KyouView />
</template>
<script setup lang="ts">
import type { EditIDFKyouDialogProps } from './edit-idf-kyou-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import EditIDFKyouView from '../views/edit-idf-kyou-view.vue';
import KyouView from '../views/kyou-view.vue';

const props = defineProps<EditIDFKyouDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
defineExpose({ show, hide })

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>
