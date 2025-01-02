<template>
    <v-row class="py-0 my-0">
        <v-col cols="auto" class="py-0 my-0 background-white">
            <ShareButton @request_open_share_mi_dialog="show_share_task_list_dialog()" />
        </v-col>
        <v-col cols="auto" class="py-0 my-0 background-white">
            <ShareTaskListDialog :application_config="application_config" :gkill_api="gkill_api"
                :find_kyou_query="find_kyou_query" @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_show_share_task_link_dialog="(share_mi_task_list_info) => show_share_task_list_link_dialog(share_mi_task_list_info)"
                ref="share_task_list_dialog" />
            <ShareTaskListLinkDialog :application_config="application_config" :gkill_api="gkill_api"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                ref="share_task_list_link_dialog" />
            <ManageShareTaskListDialog :application_config="application_config" :gkill_api="gkill_api"
                @received_errors="(errors) => emits('received_errors', errors)"
                @received_messages="(messages) => emits('received_messages', messages)"
                @requested_show_share_task_link_dialog="(share_mi_task_list_info) => show_share_task_list_link_dialog(share_mi_task_list_info)"
                ref="manage_share_task_list_dialog" />
        </v-col>
        <v-spacer class="pa-0 ma-0 background-white" />
        <v-col cols="auto" class="py-0 my-0 background-white">
            <ManageShareButton @request_open_manage_share_mi_dialog="show_manage_share_mi_dialog()" />
        </v-col>
    </v-row>
</template>
<script setup lang="ts">
import ManageShareButton from './manage-share-button.vue'
import ShareButton from './share-button.vue'
import type { miShareFooterEmits } from './mi-share-footer-emits'
import type { miShareFooterProps } from './mi-share-footer-props'
import ManageShareTaskListDialog from '../dialogs/manage-share-task-list-dialog.vue'
import ShareTaskListDialog from '../dialogs/share-task-list-dialog.vue'
import ShareTaskListLinkDialog from '../dialogs/share-task-list-link-dialog.vue'
import { ref } from 'vue'
import type { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info'

const share_task_list_dialog = ref<InstanceType<typeof ShareTaskListDialog> | null>(null);
const share_task_list_link_dialog = ref<InstanceType<typeof ShareTaskListLinkDialog> | null>(null);
const manage_share_task_list_dialog = ref<InstanceType<typeof ManageShareTaskListDialog> | null>(null);

defineProps<miShareFooterProps>()
const emits = defineEmits<miShareFooterEmits>()

function show_share_task_list_dialog() {
    const dialog = share_task_list_dialog.value
    if (dialog) {
        dialog.show()
    }
}

function show_share_task_list_link_dialog(share_mi_task_list_info: ShareMiTaskListInfo) {
    const dialog = share_task_list_link_dialog.value
    if (dialog) {
        dialog.show(share_mi_task_list_info)
    }
}

function show_manage_share_mi_dialog() {
    const dialog = manage_share_task_list_dialog.value
    if (dialog) {
        dialog.show()
    }
}

</script>
<style lang="css" scoped>
.background-white {
    background-color: white;
    z-index: 10000;
}
</style>