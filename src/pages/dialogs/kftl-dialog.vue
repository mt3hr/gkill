<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <KFTLView :app_content_height="app_content_height" :app_content_width="app_content_width.valueOf() - 50"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
            @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)" ref="kftl_view" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { nextTick, type Ref, ref } from 'vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { KFTLDialogEmits } from './kftl-dialog-emits'
import type { KFTLDialogProps } from './kftl-dialog-props'
import KFTLView from '../views/kftl-view.vue'

const kftl_view = ref<InstanceType<typeof KFTLView> | null>(null);

defineProps<KFTLDialogProps>()
const emits = defineEmits<KFTLDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(): Promise<void> {
    is_show_dialog.value = true
    nextTick(() => kftl_view.value?.focus_kftl_text_area())
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
