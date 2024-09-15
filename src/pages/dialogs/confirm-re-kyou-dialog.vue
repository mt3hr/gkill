<template>
    <ConfirmReKyouView />
</template>
<script lang="ts" setup>
import type { ConfirmReKyouDialogProps } from './confirm-re-kyou-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import ConfirmReKyouView from '../views/confirm-re-kyou-view.vue';

const props = defineProps<ConfirmReKyouDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
defineExpose({ show, hide })

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>
