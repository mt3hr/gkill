<template>
    <v-dialog v-model="is_show_dialog">
        <ConfirmDeleteRepView :application_config="application_config" :gkill_api="gkill_api" :rep_id="rep_id"
            :server_config="server_config" @requested_delete_rep="(rep) => emits('requested_delete_rep', rep)"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ConfirmDeleteRepDialogEmits } from './confirm-delete-rep-dialog-emits'
import type { ConfirmDeleteRepDialogProps } from './confirm-delete-rep-dialog-props'
import ConfirmDeleteRepView from '../views/confirm-delete-rep-view.vue'

const props = defineProps<ConfirmDeleteRepDialogProps>()
const emits = defineEmits<ConfirmDeleteRepDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
