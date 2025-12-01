<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmGenerateTLSFilesView :application_config="application_config" :gkill_api="gkill_api"
            :server_config="server_config" @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @generated_tls_files="emits('generated_tls_files')" @requested_close_dialog="hide" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { ConfirmGenerateTLSFilesDialogEmits } from './confirm-generate-tls-files-dialog-emits'
import type { ConfirmGenerateTLSFilesDialogProps } from './confirm-generate-tls-files-dialog-props'
import ConfirmGenerateTLSFilesView from '../views/confirm-generate-tls-files-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<ConfirmGenerateTLSFilesDialogProps>()
const emits = defineEmits<ConfirmGenerateTLSFilesDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
