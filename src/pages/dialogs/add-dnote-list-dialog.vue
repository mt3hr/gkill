<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <AddDnoteListView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
            @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
            @requested_add_dnote_list_query="(dnote_list_query) => emits('requested_add_dnote_list_query', dnote_list_query)"
            @requested_close_dialog="hide()" />

    </v-dialog>
</template>

<script setup lang="ts">
import AddDnoteListView from '@/pages/views/add-dnote-list-view.vue';
import { ref, type Ref } from 'vue'
import type { GkillError } from '../../classes/api/gkill-error';
import type { GkillMessage } from '../../classes/api/gkill-message';
import type AddDnoteListDialogEmits from './add-dnote-list-dialog-emits';
import type AddDnoteListDialogProps from './add-dnote-list-dialog-props';
const is_show_dialog: Ref<boolean> = ref(false)

defineExpose({ show, hide })
defineProps<AddDnoteListDialogProps>()
const emits = defineEmits<AddDnoteListDialogEmits>()

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
