<template>
    <KyouView />
    <KyouHistoriesView />
</template>
<script setup lang="ts">
import type { KyouHistoriesDialogProps } from './kyou-histories-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import KyouHistoriesView from '../views/kyou-histories-view.vue';
import KyouView from '../views/kyou-view.vue';

const props = defineProps<KyouHistoriesDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
defineExpose({ show, hide })

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>