<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <UploadFileView :app_content_height="app_content_height" :app_content_width="app_content_width"
            :last_added_tag="last_added_tag" :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>

<script lang="ts" setup>
import type { UploadFileDialogProps } from './upload-file-dialog-props'

import UploadFileView from '../views/upload-file-view.vue'
import { type Ref, ref } from 'vue'
import type { KyouViewEmits } from '../views/kyou-view-emits';

defineProps<UploadFileDialogProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
