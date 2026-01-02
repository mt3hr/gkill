<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <EditRepStructView :application_config="application_config" :gkill_api="gkill_api"
            @requested_reload_application_config="(...application_config :any[]) => emits('requested_reload_application_config', application_config[0] as ApplicationConfig)"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { EditRepStructDialogEmits } from './edit-rep-struct-dialog-emits'
import type { EditRepStructDialogProps } from './edit-rep-struct-dialog-props'
import EditRepStructView from '../views/edit-rep-struct-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { ApplicationConfig } from '@/classes/datas/config/application-config'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<EditRepStructDialogProps>()
const emits = defineEmits<EditRepStructDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
