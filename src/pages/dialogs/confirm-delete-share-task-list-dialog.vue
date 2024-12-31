<template>
    <v-dialog v-model="is_show_dialog">
        <ConfirmDeleteShareTaskListView v-if="share_mi_task_list_info" :application_config="application_config"
            :gkill_api="gkill_api" :share_mi_task_list_info="share_mi_task_list_info"
            @requested_delete_share_task_link_info="(share_task_link_info) => emits('requested_delete_share_task_link_info', share_task_link_info)"
            @requested_close_dialog="hide()"
            @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
            @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)" />
    </v-dialog>
</template>

<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ConfirmDeleteShareTaskLinkDialogEmits } from './confirm-delete-share-task-link-dialog-emits'
import type { ConfirmDeleteShareTaskLinkDialogProps } from './confirm-delete-share-task-link-dialog-props'
import ConfirmDeleteShareTaskListView from '../views/confirm-delete-share-task-list-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info'

const props = defineProps<ConfirmDeleteShareTaskLinkDialogProps>()
const emits = defineEmits<ConfirmDeleteShareTaskLinkDialogEmits>()
defineExpose({ show, hide })

const share_mi_task_list_info: Ref<ShareMiTaskListInfo | null> = ref(null)

const is_show_dialog: Ref<boolean> = ref(false)

async function show(share_mi_task_list_info_: ShareMiTaskListInfo): Promise<void> {
    share_mi_task_list_info.value = share_mi_task_list_info_
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
