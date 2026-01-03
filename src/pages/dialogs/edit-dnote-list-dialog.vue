<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <EditDnoteListView :application_config="application_config" :gkill_api="gkill_api"
            :dnote_list_query="dnote_list_query"
            @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
            @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
            @requested_update_dnote_list_query="(...dnote_list_query :any[]) => emits('requested_update_dnote_list_query', dnote_list_query[0] as DnoteListQuery)"
            @requested_close_dialog="hide()" />
    </v-dialog>
</template>

<script setup lang="ts">
import EditDnoteListView from '@/pages/views/edit-dnote-list-view.vue';
import { ref, type Ref } from 'vue'
import type { GkillError } from '../../classes/api/gkill-error';
import type { GkillMessage } from '../../classes/api/gkill-message';
import type EditDnoteListDialogEmits from './edit-dnote-list-dialog-emits';
import type EditDnoteListDialogProps from './edit-dnote-list-dialog-props';
import type DnoteListQuery from '../views/dnote-list-query';
import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

defineExpose({ show, hide })
defineProps<EditDnoteListDialogProps>()
const emits = defineEmits<EditDnoteListDialogEmits>()

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
