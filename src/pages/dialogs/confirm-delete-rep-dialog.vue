<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmDeleteRepView :application_config="application_config" :gkill_api="gkill_api"
            :repository="cloned_repository" :server_configs="server_configs"
            @requested_delete_rep="(rep) => emits('requested_delete_rep', rep)"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_close_dialog="hide()"
             />
    </v-dialog>
</template>
<script lang="ts" setup>
import { i18n } from '@/i18n'
import { type Ref, ref } from 'vue'
import type { ConfirmDeleteRepDialogEmits } from './confirm-delete-rep-dialog-emits'
import type { ConfirmDeleteRepDialogProps } from './confirm-delete-rep-dialog-props'
import ConfirmDeleteRepView from '../views/confirm-delete-rep-view.vue'
import { Repository } from '@/classes/datas/config/repository';

defineProps<ConfirmDeleteRepDialogProps>()
const emits = defineEmits<ConfirmDeleteRepDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)
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
