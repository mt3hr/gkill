<template>
    <v-dialog v-model="is_show_dialog">
        <ShareTaskListLinkView v-if="share_mi_task_list_info" :application_config="application_config"
            :gkill_api="gkill_api" :share_mi_task_list_info="share_mi_task_list_info"
            @updated_share_mi_task_list_info="(share_mi_task_list_info: ShareMiTaskListInfo) => emits('updated_share_mi_task_list_info', share_mi_task_list_info)"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @requested_close_dialog="hide()" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { computed, nextTick, type Ref, ref } from 'vue'
import type { ShareTaskListLinkDialogEmits } from './share-task-list-link-dialog-emits'
import type { ShareTaskListLinkDialogProps } from './share-task-list-link-dialog-props'
import { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info'
import ShareTaskListLinkView from '../views/share-task-list-link-view.vue'

const props = defineProps<ShareTaskListLinkDialogProps>()
const emits = defineEmits<ShareTaskListLinkDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)
const share_mi_task_list_info: Ref<ShareMiTaskListInfo | null> = ref(null)

async function show(share_mi_task_list_info_: ShareMiTaskListInfo): Promise<void> {
    share_mi_task_list_info.value = share_mi_task_list_info_
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    share_mi_task_list_info.value = null
    is_show_dialog.value = false
}
</script>
