<template>
    <AddLantanaView />
</template>
<script lang="ts" setup>
import type { LantanaDialogEmits } from './lantana-dialog-emits';
import type { LantanaDialogProps } from './lantana-dialog-props';
import type { LantanaTextData } from '@/classes/lantana/lantana-text-data';
import AddLantanaView from '../views/add-lantana-view.vue';

const props = defineProps<LantanaDialogProps>();
const emits = defineEmits<LantanaDialogEmits>();

async function add_text_data(): Promise<void> {
    // ˅
    throw new Error('Not implemented');
    // ˄
}

async function update_mood(mood: Number): Promise<void> {
    // ˅
    throw new Error('Not implemented');
    // ˄
}

async function update_text_data(text_data: LantanaTextData): Promise<void> {
    // ˅
    throw new Error('Not implemented');
    // ˄
}

async function delete_text_data(text_data: LantanaTextData): Promise<boolean> {
    // ˅
    throw new Error('Not implemented');
    // ˄
}

async function request_close_dialog(): Promise<void> {
    // ˅
    throw new Error('Not implemented');
    // ˄
}
</script>
