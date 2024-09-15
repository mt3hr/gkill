<template>
    <EditLantanaView />
    <KyouView />
</template>
<script setup lang="ts">
import type { EditLantanaDialogProps } from './edit-lantana-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import EditLantanaView from '../views/edit-lantana-view.vue';
import KyouView from '../views/kyou-view.vue';

const props = defineProps<EditLantanaDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
defineExpose({ show, hide })

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>