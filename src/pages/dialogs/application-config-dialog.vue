<template>
    <v-dialog v-model="is_show_dialog">
        <ApplicationConfigView :application_config="application_config" :gkill_api="gkill_api"
            :app_content_height="app_content_height" :app_content_width="app_content_width"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_application_config="(application_config) => emits('requested_reload_application_config', application_config)" />
    </v-dialog>
</template>

<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import { defineProps } from 'vue'
import type { ApplicationConfigDialogProps } from './application-config-dialog-props'
import type { ApplicationConfigDialogEmits } from './application-config-dialog-emits'
import ApplicationConfigView from '../views/application-config-view.vue'

const props = defineProps<ApplicationConfigDialogProps>()
const emits = defineEmits<ApplicationConfigDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
