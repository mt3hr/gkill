<template>
    <v-dialog v-model="is_show_dialog">
        <DecideRelatedTimeUploadedFileView :application_config="application_config" :gkill_api="gkill_api"
            :app_content_height="app_content_height" :app_content_width="app_content_width"
            :uploaded_kyous="uploaded_kyous" @received_errors="(errors) => emits('received_errors', errors)"
            :last_added_tag="last_added_tag" @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyou, is_checked) => emits('requested_update_check_kyous', kyou, is_checked)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { DecideRelatedTimeUploadedFileDialogEmits } from './decide-related-time-uploaded-file-dialog-emits'
import type { DecideRelatedTimeUploadedFileDialogProps } from './decide-related-time-uploaded-file-dialog-props'
import DecideRelatedTimeUploadedFileView from '../views/decide-related-time-uploaded-file-view.vue'

defineProps<DecideRelatedTimeUploadedFileDialogProps>()
const emits = defineEmits<DecideRelatedTimeUploadedFileDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
