<template>
    <EditReKyouView />
</template>
<script lang="ts" setup>
import type { EditRekyouDialogProps } from './edit-rekyou-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import EditReKyouView from '../views/edit-re-kyou-view.vue';

const props = defineProps<EditRekyouDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
defineExpose({ show, hide })

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>
