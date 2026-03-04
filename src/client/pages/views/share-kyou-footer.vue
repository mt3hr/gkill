<template>
    <v-row class="py-0 my-0">
        <v-col cols="auto" class="py-0 my-0 background-white">
            <ShareButton @request_open_share_kyou_dialog="show_share_kyou_list_dialog()" />
        </v-col>
        <v-col cols="auto" class="py-0 my-0 background-white">
            <ShareKyousListDialog :application_config="application_config" :gkill_api="gkill_api"
                :find_kyou_query="find_kyou_query"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_show_share_kyou_link_dialog="(...share_kyou_list_info: any[]) => show_share_kyou_list_link_dialog(share_kyou_list_info[0] as ShareKyousInfo)"
                ref="share_kyou_list_dialog" />
            <ShareKyousListLinkDialog :application_config="application_config" :gkill_api="gkill_api"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                ref="share_kyou_list_link_dialog" />
            <ManageShareKyousListDialog :application_config="application_config" :gkill_api="gkill_api"
                @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
                @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)"
                @requested_show_share_kyou_link_dialog="(...share_kyou_list_info: any[]) => show_share_kyou_list_link_dialog(share_kyou_list_info[0] as ShareKyousInfo)"
                ref="manage_share_kyou_list_dialog" />
        </v-col>
        <v-spacer class="pa-0 ma-0 background-white" />
        <v-col cols="auto" class="py-0 my-0 background-white">
            <ManageShareButton @request_open_manage_share_kyou_dialog="show_manage_share_kyou_dialog()" />
        </v-col>
    </v-row>
</template>
<script setup lang="ts">
import ManageShareButton from './manage-share-button.vue'
import ShareButton from './share-button.vue'
import type { ShareKyouFooterEmits } from './share-kyou-footer-emits'
import type { ShareKyouFooterProps } from './share-kyou-footer-props'
import ManageShareKyousListDialog from '../dialogs/manage-share-task-list-dialog.vue'
import ShareKyousListDialog from '../dialogs/share-kyou-list-dialog.vue'
import ShareKyousListLinkDialog from '../dialogs/share-kyou-list-link-dialog.vue'
import { ref } from 'vue'
import type { ShareKyousInfo } from '@/classes/datas/share-kyous-info'
import type { GkillError } from '@/classes/api/gkill-error'
import type { GkillMessage } from '@/classes/api/gkill-message'

const share_kyou_list_dialog = ref<InstanceType<typeof ShareKyousListDialog> | null>(null);
const share_kyou_list_link_dialog = ref<InstanceType<typeof ShareKyousListLinkDialog> | null>(null);
const manage_share_kyou_list_dialog = ref<InstanceType<typeof ManageShareKyousListDialog> | null>(null);

defineProps<ShareKyouFooterProps>()
const emits = defineEmits<ShareKyouFooterEmits>()

function show_share_kyou_list_dialog() {
    const dialog = share_kyou_list_dialog.value
    if (dialog) {
        dialog.show()
    }
}

function show_share_kyou_list_link_dialog(share_kyou_list_info: ShareKyousInfo) {
    const dialog = share_kyou_list_link_dialog.value
    if (dialog) {
        dialog.show(share_kyou_list_info)
    }
}

function show_manage_share_kyou_dialog() {
    const dialog = manage_share_kyou_list_dialog.value
    if (dialog) {
        dialog.show()
    }
}

</script>
<style lang="css" scoped>
.background-white {
    background-color: var(--v-primary-base);
    z-index: 10000;
}
</style>