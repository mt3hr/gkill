<template>
    <EditKmemoView />
    <KyouView />
</template>
<script setup lang="ts">
import type { EditKmemoDialogProps } from './edit-kmemo-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import EditKmemoView from '../views/edit-kmemo-view.vue';
import KyouView from '../views/kyou-view.vue';

const props = defineProps<EditKmemoDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
defineExpose({ show, hide })

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}

</script>
