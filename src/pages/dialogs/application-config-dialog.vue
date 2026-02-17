<template>
    <v-dialog persistent @click:outside="hide" @keydown.esc="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <ApplicationConfigView :application_config="application_config" :gkill_api="gkill_api"
            :app_content_height="app_content_height" :app_content_width="app_content_width"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_close_dialog="() => hide()" ref="application_config_view" />
    </v-dialog>
</template>

<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import { defineProps } from 'vue'
import type { ApplicationConfigDialogProps } from './application-config-dialog-props'
import type { ApplicationConfigDialogEmits } from './application-config-dialog-emits'
import ApplicationConfigView from '../views/application-config-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import { GkillMessage } from '@/classes/api/gkill-message'

const application_config_view = ref<InstanceType<typeof ApplicationConfigView> | null>(null);

const _props = defineProps<ApplicationConfigDialogProps>()
const emits = defineEmits<ApplicationConfigDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(): Promise<void> {
    is_show_dialog.value = true
    application_config_view.value?.reload_cloned_application_config()
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
