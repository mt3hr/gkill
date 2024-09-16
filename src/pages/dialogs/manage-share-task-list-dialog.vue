<template>
    <v-dialog v-model="is_show_dialog">
        <ManageShareTaskListView :application_config="application_config" :gkill_api="gkill_api"
            :share_mi_task_list_infos="share_mi_task_list_infos"
            @requested_show_confirm_delete_share_task_list_dialog="show_confirm_delete_share_task_list_dialog"
            @requested_show_share_task_link_dialog="show_share_task_list_link_dialog" />
        <ShareTaskListLinkDialog :share_mi_task_list_info="share_task_list" :application_config="application_config"
            :gkill_api="gkill_api" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" ref="share_task_list_link_dialog" />
        <ConfirmDeleteShareTaskListDialog :share_mi_task_list_info="share_task_link"
            :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            ref="confirm_delete_share_task_list_dialog" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { ManageShareTaskLinkDialogEmits } from './manage-share-task-link-dialog-emits';
import type { ManageShareTaskLinkDialogProps } from './manage-share-task-link-dialog-props';
import ManageShareTaskListView from '../views/manage-share-task-list-view.vue';
import ShareTaskListLinkDialog from './share-task-list-link-dialog.vue';
import ConfirmDeleteShareTaskListDialog from './confirm-delete-share-task-list-dialog.vue';
import { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info';

const share_task_list_link_dialog = ref<InstanceType<typeof ShareTaskListLinkDialog> | null>(null);
const confirm_delete_share_task_list_dialog = ref<InstanceType<typeof ConfirmDeleteShareTaskListDialog> | null>(null);

const props = defineProps<ManageShareTaskLinkDialogProps>();
const emits = defineEmits<ManageShareTaskLinkDialogEmits>();
defineExpose({ show, hide })

const cloned_share_mi_task_list_infos: Ref<Array<ShareMiTaskListInfo>> = ref(new Array<ShareMiTaskListInfo>());
const share_task_list: Ref<ShareMiTaskListInfo> = ref(new ShareMiTaskListInfo())
const share_task_link: Ref<ShareMiTaskListInfo> = ref(new ShareMiTaskListInfo())

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
async function show_confirm_delete_share_task_list_dialog(share_mi_task_list_info: ShareMiTaskListInfo): Promise<void> {
    share_task_list.value = share_mi_task_list_info
    confirm_delete_share_task_list_dialog.value?.show()
}
async function show_share_task_list_link_dialog(share_mi_task_list_info: ShareMiTaskListInfo): Promise<void> {
    share_task_link.value = share_mi_task_list_info
    share_task_list_link_dialog.value?.show()
}
</script>
