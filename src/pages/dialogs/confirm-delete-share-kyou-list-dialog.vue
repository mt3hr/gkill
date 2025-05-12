<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmDeleteShareKyousListView v-if="share_kyou_list_info" :application_config="application_config"
            :gkill_api="gkill_api" :share_kyou_list_info="share_kyou_list_info"
            @requested_delete_share_kyou_link_info="(share_kyou_link_info) => emits('requested_delete_share_kyou_link_info', share_kyou_link_info)"
            @requested_close_dialog="hide()"
            @received_errors="(errors: GkillError[]) => emits('received_errors', errors)"
            @received_messages="(messages: GkillMessage[]) => emits('received_messages', messages)" />
    </v-dialog>
</template>

<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ConfirmDeleteShareKyousLinkDialogEmits } from './confirm-delete-share-kyou-link-dialog-emits'
import type { ConfirmDeleteShareKyousLinkDialogProps } from './confirm-delete-share-kyou-link-dialog-props'
import ConfirmDeleteShareKyousListView from '../views/confirm-delete-share-task-list-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'
import type { ShareKyouListInfo } from '@/classes/datas/share-kyou-list-info'

defineProps<ConfirmDeleteShareKyousLinkDialogProps>()
const emits = defineEmits<ConfirmDeleteShareKyousLinkDialogEmits>()
defineExpose({ show, hide })

const share_kyou_list_info: Ref<ShareKyouListInfo | null> = ref(null)

const is_show_dialog: Ref<boolean> = ref(false)

async function show(share_kyou_list_info_: ShareKyouListInfo): Promise<void> {
    share_kyou_list_info.value = share_kyou_list_info_
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
