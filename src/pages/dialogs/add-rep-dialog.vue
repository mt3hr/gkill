<template>
    <v-dialog v-model="is_show_dialog">
        <AddRepView :application_config="application_config" :gkill_api="gkill_api" :server_config="server_config"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_add_rep="(rep) => emits('requested_add_rep', rep)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { AddRepDialogEmits } from './add-rep-dialog-emits'
import type { AddRepDialogProps } from './add-rep-dialog-props'
import AddRepView from '../views/add-rep-view.vue'

const props = defineProps<AddRepDialogProps>()
const emits = defineEmits<AddRepDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
