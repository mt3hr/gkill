<template>
    <v-dialog v-model="is_show_dialog">
        <ShareTaskListLinkView :application_config="application_config" :gkill_api="gkill_api"
            :share_mi_task_list_info="share_mi_task_list_info"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ShareTaskListLinkDialogEmits } from './share-task-list-link-dialog-emits'
import type { ShareTaskListLinkDialogProps } from './share-task-list-link-dialog-props'
import type { ShareMiTaskListInfo } from '@/classes/datas/share-mi-task-list-info'
import ShareTaskListLinkView from '../views/share-task-list-link-view.vue'

const props = defineProps<ShareTaskListLinkDialogProps>()
const emits = defineEmits<ShareTaskListLinkDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
const cloned_share_mi_task_list_info: Ref<ShareMiTaskListInfo> = ref(await props.share_mi_task_list_info.clone())
const is_share_time_only: Ref<boolean> = ref(!cloned_share_mi_task_list_info.value.is_share_detail)
</script>
