<template>
    <v-dialog persistent @click:outside="hide" @keydown.esc="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <UploadFileView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :last_added_tag="last_added_tag" :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" />
    </v-dialog>
</template>

<script lang="ts" setup>
import type { UploadFileDialogProps } from './upload-file-dialog-props'

import UploadFileView from '../views/upload-file-view.vue'
import { type Ref, ref } from 'vue'
import type { KyouViewEmits } from '../views/kyou-view-emits';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

defineProps<UploadFileDialogProps>()
const emits = defineEmits<KyouViewEmits>()
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
