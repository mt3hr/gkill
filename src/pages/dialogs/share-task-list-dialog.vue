<template>
    <v-dialog v-model="is_show_dialog">
        <ShareTaskListView :application_config="application_config" :gkill_api="gkill_api"
            :find_kyou_query="find_kyou_query" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @regestered_share_mi_task_list_info="(share_mi_task_info) => {
                emits('regestered_share_mi_task_list_info', share_mi_task_info)
                emits('requested_show_share_task_link_dialog', share_mi_task_info)
            }" @requested_close_dialog="hide()" />
    </v-dialog>
</template>
<script lang="ts" setup>
import type { ShareTaskListDialogEmits } from './share-task-list-dialog-emits'
import type { ShareTaskListDialogProps } from './share-task-list-dialog-props'
import ShareTaskListView from '../views/share-task-list-view.vue'
import { ref, type Ref } from 'vue'

const props = defineProps<ShareTaskListDialogProps>()
const emits = defineEmits<ShareTaskListDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
