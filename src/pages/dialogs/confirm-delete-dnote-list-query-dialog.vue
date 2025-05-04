<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <ConfirmDeleteDnoteListQueryView :application_config="application_config" :gkill_api="gkill_api"
            :dnote_list_query="dnote_list_query" @received_errors="(errors) => emits('received_errors', errors)"
            @requested_close_dialog="hide"
            @requested_delete_dnote_list_query="(id) => { emits('requested_delete_dnote_list_query', id); hide() }"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import DnoteListQuery from '../views/dnote-list-query';
import ConfirmDeleteDnoteListQueryView from '../views/confirm-delete-dnote-list-query-view.vue';
import type { ConfirmDeleteDnoteListQueryDialogEmits } from './confirm-delete-dnote-list-query-dialog-emits';
import type { ConfirmDeleteDnoteListQueryDialogProps } from './confirm-delete-dnote-list-query-dialog-props';

defineProps<ConfirmDeleteDnoteListQueryDialogProps>()
const emits = defineEmits<ConfirmDeleteDnoteListQueryDialogEmits>()
defineExpose({ show, hide })

const dnote_list_query: Ref<DnoteListQuery> = ref(new DnoteListQuery())

const is_show_dialog: Ref<boolean> = ref(false)

async function show(_dnote_item: DnoteListQuery): Promise<void> {
    dnote_list_query.value = _dnote_item
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
    dnote_list_query.value = new DnoteListQuery()
}
</script>
