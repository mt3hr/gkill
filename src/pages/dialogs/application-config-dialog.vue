<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ApplicationConfigView :application_config="application_config" :gkill_api="gkill_api"
            :app_content_height="app_content_height" :app_content_width="app_content_width"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_application_config="emits('requested_reload_application_config', application_config)"
            @requested_close_dialog="hide" ref="application_config_view" />
    </v-dialog>
</template>

<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import { defineProps } from 'vue'
import type { ApplicationConfigDialogProps } from './application-config-dialog-props'
import type { ApplicationConfigDialogEmits } from './application-config-dialog-emits'
import ApplicationConfigView from '../views/application-config-view.vue'

const application_config_view = ref<InstanceType<typeof ApplicationConfigView> | null>(null);

defineProps<ApplicationConfigDialogProps>()
const emits = defineEmits<ApplicationConfigDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
    application_config_view.value?.reload_cloned_application_config()
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
