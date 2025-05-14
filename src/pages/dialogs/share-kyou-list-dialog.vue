<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ShareKyousListView :application_config="application_config" :gkill_api="gkill_api"
            :find_kyou_query="find_kyou_query" @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" @regestered_share_kyou_list_info="(share_kyou_info) => {
                emits('regestered_share_kyou_list_info', share_kyou_info)
                emits('requested_show_share_kyou_link_dialog', share_kyou_info)
            }" @requested_close_dialog="hide()" />
    </v-dialog>
</template>
<script lang="ts" setup>
import type { ShareKyousListDialogEmits } from './share-kyou-list-dialog-emits'
import type { ShareKyousListDialogProps } from './share-kyou-list-dialog-props'
import ShareKyousListView from '../views/share-kyou-view.vue'
import { ref, type Ref } from 'vue'

defineProps<ShareKyousListDialogProps>()
const emits = defineEmits<ShareKyousListDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
