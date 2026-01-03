<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ShareKyousListLinkView v-if="share_kyou_list_info" :application_config="application_config"
            :gkill_api="gkill_api" :share_kyou_list_info="share_kyou_list_info"
            @updated_share_kyou_list_info="(share_kyou_list_info: ShareKyousInfo) => emits('updated_share_kyou_list_info', share_kyou_list_info)"
            @received_errors="(...errors :any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @received_messages="(...messages :any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" @requested_close_dialog="hide()" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ShareKyousListLinkDialogEmits } from './share-kyou-list-link-dialog-emits'
import type { ShareKyousListLinkDialogProps } from './share-kyou-list-link-dialog-props'
import { ShareKyousInfo } from '@/classes/datas/share-kyous-info'
import ShareKyousListLinkView from '../views/share-kyou-link-view.vue'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

defineProps<ShareKyousListLinkDialogProps>()
const emits = defineEmits<ShareKyousListLinkDialogEmits>()
defineExpose({ show, hide })

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)
const share_kyou_list_info: Ref<ShareKyousInfo | null> = ref(null)

async function show(share_kyou_list_info_: ShareKyousInfo): Promise<void> {
    share_kyou_list_info.value = share_kyou_list_info_
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    share_kyou_list_info.value = null
    is_show_dialog.value = false
}
</script>
