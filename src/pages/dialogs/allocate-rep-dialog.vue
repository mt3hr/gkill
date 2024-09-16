<template>
    <v-dialog v-model="is_show_dialog">
        <AllocateRepView :application_config="application_config" :gkill_api="gkill_api" :server_config="server_config"
            :user_id="user_id" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_server_config="(server_config) => emits('requested_reload_server_config', server_config)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { AllocateRepDialogEmits } from './allocate-rep-dialog-emits';
import type { AllocateRepDialogProps } from './allocate-rep-dialog-props';
import AllocateRepView from '../views/allocate-rep-view.vue';

const props = defineProps<AllocateRepDialogProps>();
const emits = defineEmits<AllocateRepDialogEmits>();
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
