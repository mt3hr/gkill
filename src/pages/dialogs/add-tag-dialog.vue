<template>
    <AddTagView />
    <KyouView />
</template>
<script setup lang="ts">
import type { AddTagDialogProps } from './add-tag-dialog-props';
import type { KyouDialogEmits } from '../views/kyou-dialog-emits';
import AddTagView from '../views/add-tag-view.vue';
import KyouView from '../views/kyou-view.vue';

const props = defineProps<AddTagDialogProps>();
const emits = defineEmits<KyouDialogEmits>();

async function show(): Promise<void> {
    throw new Error('Not implemented');
}
async function hide(): Promise<void> {
    throw new Error('Not implemented');
}
</script>
