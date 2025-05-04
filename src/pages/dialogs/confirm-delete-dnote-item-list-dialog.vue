<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmDeleteDnoteItemListView :application_config="application_config" :gkill_api="gkill_api"
            :dnote_item="dnote_item" @received_errors="(errors) => emits('received_errors', errors)"
            @requested_close_dialog="hide"
            @requested_delete_dnote_item_list="(id) => { emits('requested_delete_dnote_list_item', id); hide() }"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import ConfirmDeleteDnoteItemListView from '../views/confirm-delete-dnote-item-list-view.vue';
import type { ConfirmDeleteDnoteItemListDialogEmits } from './confirm-delete-dnote-item-list-dialog-emits';
import type { ConfirmDeleteDnoteItemListDialogProps } from './confirm-delete-dnote-item-list-dialog-props';
import DnoteItem from '@/classes/dnote/dnote-item';

defineProps<ConfirmDeleteDnoteItemListDialogProps>()
const emits = defineEmits<ConfirmDeleteDnoteItemListDialogEmits>()
defineExpose({ show, hide })

const dnote_item: Ref<DnoteItem> = ref(new DnoteItem())

const is_show_dialog: Ref<boolean> = ref(false)

async function show(_dnote_item: DnoteItem): Promise<void> {
    dnote_item.value = _dnote_item
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    dnote_item.value = new DnoteItem()
}
</script>
