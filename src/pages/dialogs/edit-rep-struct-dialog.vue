<template>
    <v-dialog v-model="is_show_dialog">
        <EditRepStructView :application_config="application_config" :gkill_api="gkill_api"
            @requested_reload_application_config="(application_config) => emits('requested_reload_application_config', application_config)"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditRepStructDialogEmits } from './edit-rep-struct-dialog-emits'
import type { EditRepStructDialogProps } from './edit-rep-struct-dialog-props'
import EditRepStructView from '../views/edit-rep-struct-view.vue'

const props = defineProps<EditRepStructDialogProps>()
const emits = defineEmits<EditRepStructDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
