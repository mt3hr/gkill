<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog" class="mkfl_dialog">
        <MKFLView :app_content_height="app_content_height" :app_content_width="app_content_width.valueOf() - 50"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
            @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { MKFLDialogEmits } from './mkfl-dialog-emits'
import type { MKFLDialogProps } from './mkfl-dialog-props'
import MKFLView from '../views/mkfl-view.vue'

defineProps<MKFLDialogProps>()
const emits = defineEmits<MKFLDialogEmits>()
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
<style lang="css">
.mkfl_dialog.v-dialog .v-card {
    overflow-y: hidden;
}

.mkfl_dialog.v-dialog .v-card::-webkit-scrollbar {
    width: 0px;
}
</style>