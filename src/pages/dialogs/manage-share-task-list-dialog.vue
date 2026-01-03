<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ManageShareKyousListView :application_config="application_config" :gkill_api="gkill_api"
            :share_kyou_list_infos="share_kyou_list_infos"
            @requested_show_confirm_delete_share_kyou_list_dialog="show_confirm_delete_share_kyou_list_dialog"
            @requested_show_share_kyou_link_dialog="show_share_kyou_list_link_dialog" />
        <ShareKyousListLinkDialog :share_kyou_list_info="share_kyou_list" :application_config="application_config"
            :gkill_api="gkill_api" @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @updated_share_kyou_list_info="reload_share_kyou_list_infos()"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" ref="share_kyou_list_link_dialog" />
        <ConfirmDeleteShareKyousListDialog :share_kyou_list_info="share_kyou_link"
            :application_config="application_config" :gkill_api="gkill_api"
            @requested_delete_share_kyou_link_info="(share_kyou_list_infos: ShareKyousInfo) => delete_share_kyou_link_info(share_kyou_list_infos)"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
            ref="confirm_delete_share_kyou_list_dialog" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ManageShareKyousLinkDialogEmits } from './manage-share-task-link-dialog-emits'
import type { ManageShareKyousLinkDialogProps } from './manage-share-task-link-dialog-props'
import ManageShareKyousListView from '../views/manage-share-task-list-view.vue'
import ShareKyousListLinkDialog from './share-kyou-list-link-dialog.vue'
import ConfirmDeleteShareKyousListDialog from './confirm-delete-share-kyou-list-dialog.vue'
import { ShareKyousInfo } from '@/classes/datas/share-kyous-info'
import { DeleteShareKyouListInfosRequest } from '@/classes/api/req_res/delete-share-kyou-list-infos-request'
import { GetShareKyouListInfosRequest } from '@/classes/api/req_res/get-share-kyou-list-infos-request'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

const share_kyou_list_link_dialog = ref<InstanceType<typeof ShareKyousListLinkDialog> | null>(null)
const confirm_delete_share_kyou_list_dialog = ref<InstanceType<typeof ConfirmDeleteShareKyousListDialog> | null>(null)

const share_kyou_list_infos: Ref<Array<ShareKyousInfo>> = ref(new Array<ShareKyousInfo>())

const props = defineProps<ManageShareKyousLinkDialogProps>()
const emits = defineEmits<ManageShareKyousLinkDialogEmits>()
defineExpose({ show, hide })

const share_kyou_list: Ref<ShareKyousInfo> = ref(new ShareKyousInfo())
const share_kyou_link: Ref<ShareKyousInfo> = ref(new ShareKyousInfo())

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(): Promise<void> {
    reload_share_kyou_list_infos()
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    share_kyou_list_infos.value.splice(0)
    is_show_dialog.value = false
}
async function show_confirm_delete_share_kyou_list_dialog(share_kyou_list_info: ShareKyousInfo): Promise<void> {
    share_kyou_list.value = share_kyou_list_info
    confirm_delete_share_kyou_list_dialog.value?.show(share_kyou_list_info)
}
async function show_share_kyou_list_link_dialog(share_kyou_list_info: ShareKyousInfo): Promise<void> {
    share_kyou_link.value = share_kyou_list_info
    share_kyou_list_link_dialog.value?.show(share_kyou_list_info)
}

async function delete_share_kyou_link_info(share_kyou_list_info: ShareKyousInfo): Promise<void> {
    const req = new DeleteShareKyouListInfosRequest()
    req.share_kyou_list_info = share_kyou_list_info
    const res = await props.gkill_api.delete_share_kyou_list_infos(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    await reload_share_kyou_list_infos()
}

async function reload_share_kyou_list_infos(): Promise<void> {
    const req = new GetShareKyouListInfosRequest()
    const res = await props.gkill_api.get_share_kyou_list_infos(req)
    if (res.errors && res.errors.length !== 0) {
        emits('received_errors', res.errors)
        return
    }
    if (res.messages && res.messages.length !== 0) {
        emits('received_messages', res.messages)
    }

    share_kyou_list_infos.value = res.share_kyou_list_infos
}

</script>
