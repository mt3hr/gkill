<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <DecideRelatedTimeUploadedFileView :application_config="application_config" :gkill_api="gkill_api"
            :app_content_height="app_content_height" :app_content_width="app_content_width"
            :uploaded_kyous="uploaded_kyous" @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            :last_added_tag="last_added_tag" @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_reload_kyou="(...kyou: any[]) => emits('requested_reload_kyou', kyou[0] as Kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(...params: any[]) => emits('requested_update_check_kyous', params[0] as Array<Kyou>, params[1] as boolean)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { DecideRelatedTimeUploadedFileDialogEmits } from './decide-related-time-uploaded-file-dialog-emits'
import type { DecideRelatedTimeUploadedFileDialogProps } from './decide-related-time-uploaded-file-dialog-props'
import DecideRelatedTimeUploadedFileView from '../views/decide-related-time-uploaded-file-view.vue'
import type { Kyou } from '@/classes/datas/kyou'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<DecideRelatedTimeUploadedFileDialogProps>()
const emits = defineEmits<DecideRelatedTimeUploadedFileDialogEmits>()
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
