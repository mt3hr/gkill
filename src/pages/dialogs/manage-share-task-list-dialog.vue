<template>
    <v-dialog v-model="is_show_dialog">
        <ManageShareTaskListView :application_config="application_config" :gkill_api="gkill_api"
            :share_mi_task_list_infos="share_mi_task_list_infos"
            @requested_show_confirm_delete_share_task_list_dialog="show_confirm_delete_share_task_list_dialog"
            @requested_show_share_task_link_dialog="show_share_task_list_link_dialog" />
        <ShareTaskListLinkDialog :share_mi_task_list_info="share_task_list" :application_config="application_config"
            :gkill_api="gkill_api" @received_errors="(errors) => emits('received_errors', errors)"
            @updated_share_mi_task_list_info="reload_share_mi_task_list_infos()"
            @received_messages="(messages) => emits('received_messages', messages)" ref="share_task_list_link_dialog" />
        <ConfirmDeleteShareTaskListDialog :share_mi_task_list_info="share_task_link"
            :application_config="application_config" :gkill_api="gkill_api"
            @requested_delete_share_task_link_info="(share_mi_task_list_infos: ShareMiTaskListInfo) => delete_share_task_link_info(share_mi_task_list_infos)"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            ref="confirm_delete_share_task_list_dialog" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ManageShareTaskLinkDialogEmits } from './manage-share-task-link-dialog-emits'
import type { ManageShareTaskLinkDialogProps } from './manage-share-task-link-dialog-props'
import ManageShareTaskListView from '../views/manage-share-task-list-view.vue'
import ShareTaskListLinkDialog from './share-task-list-link-dialog.vue'
import ConfirmDeleteShareTaskListDialog from './confirm-delete-share-task-list-dialog.vue'
import { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info'
import { DeleteShareMiTaskListInfosRequest } from '@/classes/api/req_res/delete-share-mi-task-list-infos-request'
import { GkillAPI } from '@/classes/api/gkill-api'
import { GetShareMiTaskListInfosRequest } from '@/classes/api/req_res/get-share-mi-task-list-infos-request'

const share_task_list_link_dialog = ref<InstanceType<typeof ShareTaskListLinkDialog> | null>(null)
const confirm_delete_share_task_list_dialog = ref<InstanceType<typeof ConfirmDeleteShareTaskListDialog> | null>(null)

const share_mi_task_list_infos: Ref<Array<ShareMiTaskListInfo>> = ref(new Array<ShareMiTaskListInfo>())

const props = defineProps<ManageShareTaskLinkDialogProps>()
const emits = defineEmits<ManageShareTaskLinkDialogEmits>()
defineExpose({ show, hide })

const share_task_list: Ref<ShareMiTaskListInfo> = ref(new ShareMiTaskListInfo())
const share_task_link: Ref<ShareMiTaskListInfo> = ref(new ShareMiTaskListInfo())

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    reload_share_mi_task_list_infos()
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    share_mi_task_list_infos.value.splice(0)
    is_show_dialog.value = false
}
async function show_confirm_delete_share_task_list_dialog(share_mi_task_list_info: ShareMiTaskListInfo): Promise<void> {
    share_task_list.value = share_mi_task_list_info
    confirm_delete_share_task_list_dialog.value?.show(share_mi_task_list_info)
}
async function show_share_task_list_link_dialog(share_mi_task_list_info: ShareMiTaskListInfo): Promise<void> {
    share_task_link.value = share_mi_task_list_info
    share_task_list_link_dialog.value?.show(share_mi_task_list_info)
}

async function delete_share_task_link_info(share_mi_task_list_info: ShareMiTaskListInfo): Promise<void> {
    const req = new DeleteShareMiTaskListInfosRequest()
    req.session_id = props.gkill_api.get_session_id()
    req.share_mi_task_list_info = share_mi_task_list_info
    const res = await props.gkill_api.delete_share_mi_task_list_infos(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    await reload_share_mi_task_list_infos()
}

async function reload_share_mi_task_list_infos(): Promise<void> {
    const req = new GetShareMiTaskListInfosRequest()
    req.session_id = props.gkill_api.get_session_id()
    const res = await props.gkill_api.get_share_mi_task_list_infos(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    share_mi_task_list_infos.value = res.share_mi_task_list_infos
}

</script>
