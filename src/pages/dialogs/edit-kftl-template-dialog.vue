<template>
    <v-dialog v-model="is_show_dialog">
        <EditKFTLTemplateView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_application_config="(application_config) => emits('requested_reload_application_config', application_config)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { EditKFTLTemplateDialogEmits } from './edit-kftl-template-dialog-emits'
import type { EditKFTLTemplateDialogProps } from './edit-kftl-template-dialog-props'
import EditKFTLTemplateView from '../views/edit-kftl-template-view.vue'
const props = defineProps<EditKFTLTemplateDialogProps>()
const emits = defineEmits<EditKFTLTemplateDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
