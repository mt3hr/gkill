<template>
    <v-dialog persistent @click:outside="hide" @keydown.esc="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmDeleteRepView :application_config="application_config" :gkill_api="gkill_api"
            :repository="cloned_repository" :server_configs="server_configs"
            @requested_delete_rep="(...rep :any[]) => emits('requested_delete_rep', rep[0] as Repository)"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            @requested_close_dialog="hide()"
             />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ConfirmDeleteRepDialogEmits } from './confirm-delete-rep-dialog-emits'
import type { ConfirmDeleteRepDialogProps } from './confirm-delete-rep-dialog-props'
import ConfirmDeleteRepView from '../views/confirm-delete-rep-view.vue'
import { Repository } from '@/classes/datas/config/repository';
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<ConfirmDeleteRepDialogProps>()
const emits = defineEmits<ConfirmDeleteRepDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
const cloned_repository: Ref<Repository> = ref(new Repository())

async function show(repository: Repository): Promise<void> {
    cloned_repository.value = repository
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    cloned_repository.value = new Repository()
}
</script>
