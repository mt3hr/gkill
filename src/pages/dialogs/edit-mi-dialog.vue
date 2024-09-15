<template>
    <EditMiView />
    <KyouView />
    <NewBoardNameDialog />
</template>
<script setup lang="ts">
import type { EditMiDialogProps } from './edit-mi-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import EditMiView from '../views/edit-mi-view.vue';
import KyouView from '../views/kyou-view.vue';
import NewBoardNameDialog from './new-board-name-dialog.vue';

const props = defineProps<EditMiDialogProps>();
const emits = defineEmits<KyouDialogEmits>();
defineExpose({ show, hide })

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>
