<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <AddDnoteItemView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
            @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
            @requested_add_dnote_item="(...dnote_item :any[]) => emits('requested_add_dnote_item', dnote_item[0] as DnoteItem)"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'
import { ref, type Ref } from 'vue'
import type { GkillError } from '../../classes/api/gkill-error';
import type { GkillMessage } from '../../classes/api/gkill-message';
import type AddDnoteItemDialogEmits from './add-dnote-item-dialog-emits';
import type AddDnoteItemDialogProps from './add-dnote-item-dialog-props';
import AddDnoteItemView from '@/pages/views/add-dnote-item-view.vue';
import type DnoteItem from '@/classes/dnote/dnote-item';
import { useBackToCloseDialog } from '@/classes/use-back-to-close-dialog'
const is_show_dialog: Ref<boolean> = ref(false)
useBackToCloseDialog(is_show_dialog)

defineExpose({ show, hide })
defineProps<AddDnoteItemDialogProps>()
const emits = defineEmits<AddDnoteItemDialogEmits>()

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
