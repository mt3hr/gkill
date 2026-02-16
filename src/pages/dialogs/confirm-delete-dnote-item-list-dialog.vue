<template>
    <v-dialog persistent @click:outside="hide" :no-click-animation="true"  :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmDeleteDnoteItemListView :application_config="application_config" :gkill_api="gkill_api"
            :dnote_item="dnote_item"
            @received_errors="(...errors: any[]) => emits('received_errors', errors[0] as Array<GkillError>)"
            @requested_close_dialog="hide"
            @requested_delete_dnote_item_list="(...id: any[]) => { emits('requested_delete_dnote_list_item', id[0] as string); hide() }"
            @received_messages="(...messages: any[]) => emits('received_messages', messages[0] as Array<GkillMessage>)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import ConfirmDeleteDnoteItemListView from '../views/confirm-delete-dnote-item-list-view.vue';
import type { ConfirmDeleteDnoteItemListDialogEmits } from './confirm-delete-dnote-item-list-dialog-emits';
import type { ConfirmDeleteDnoteItemListDialogProps } from './confirm-delete-dnote-item-list-dialog-props';
import DnoteItem from '@/classes/dnote/dnote-item';
import type { GkillError } from '@/classes/api/gkill-error';
import type { GkillMessage } from '@/classes/api/gkill-message';

defineProps<ConfirmDeleteDnoteItemListDialogProps>()
const emits = defineEmits<ConfirmDeleteDnoteItemListDialogEmits>()
defineExpose({ show, hide })

const dnote_item: Ref<DnoteItem> = ref(new DnoteItem())

import { useDialogHistoryStack } from '@/classes/use-dialog-history-stack'
const is_show_dialog: Ref<boolean> = ref(false)
useDialogHistoryStack(is_show_dialog)

async function show(_dnote_item: DnoteItem): Promise<void> {
    dnote_item.value = _dnote_item
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    dnote_item.value = new DnoteItem()
}
</script>
