<template>
    <v-dialog v-model="is_show_dialog">
        <AddLantanaView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { LantanaDialogEmits } from './lantana-dialog-emits';
import type { LantanaDialogProps } from './lantana-dialog-props';
import type { LantanaTextData } from '@/classes/lantana/lantana-text-data';
import AddLantanaView from '../views/add-lantana-view.vue';

const props = defineProps<LantanaDialogProps>();
const emits = defineEmits<LantanaDialogEmits>();
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}

async function add_text_data(): Promise<void> {
    throw new Error('Not implemented');
}

async function update_mood(mood: Number): Promise<void> {
    throw new Error('Not implemented');
}

async function update_text_data(text_data: LantanaTextData): Promise<void> {
    throw new Error('Not implemented');
}

async function delete_text_data(text_data: LantanaTextData): Promise<boolean> {
    throw new Error('Not implemented');
}

async function request_close_dialog(): Promise<void> {
    throw new Error('Not implemented');
}
</script>
