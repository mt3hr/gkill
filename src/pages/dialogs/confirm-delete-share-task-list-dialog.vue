<template>
    <v-dialog v-model="is_show_dialog">
        <ConfirmDeleteShareTaskListView :application_config="application_config" :gkill_api="gkill_api"
            :share_mi_task_list_info="share_mi_task_list_info"
            @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
            @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)" />
    </v-dialog>
</template>

<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { ConfirmDeleteShareTaskLinkDialogEmits } from './confirm-delete-share-task-link-dialog-emits';
import type { ConfirmDeleteShareTaskLinkDialogProps } from './confirm-delete-share-task-link-dialog-props';
import type { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info';
import ConfirmDeleteShareTaskListView from '../views/confirm-delete-share-task-list-view.vue';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

const props = defineProps<ConfirmDeleteShareTaskLinkDialogProps>();
const emits = defineEmits<ConfirmDeleteShareTaskLinkDialogEmits>();
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)
const cloned_share_mi_task_list_info: Ref<ShareMiTaskListInfo> = ref(await props.share_mi_task_list_info.clone());

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
