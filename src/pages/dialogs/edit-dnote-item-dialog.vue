<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <EditDnoteItemView :application_config="application_config" :gkill_api="gkill_api" v-model="model_value"
            @received_errors="(errors: Array<GkillError>) => emits('received_errors', errors)"
            @received_messages="(messages: Array<GkillMessage>) => emits('received_messages', messages)"
            @requested_update_dnote_item="(...dnote_item: any[]) => emits('requested_update_dnote_item', dnote_item[0] as DnoteItem)"
            @requested_close_dialog="hide" />
    </v-dialog>
</template>

<script setup lang="ts">
import { i18n } from '@/i18n'
import EditDnoteItemView from '@/pages/views/edit-dnote-item-view.vue';
import { ref, type Ref } from 'vue'
import type { GkillError } from '../../classes/api/gkill-error';
import type { GkillMessage } from '../../classes/api/gkill-message';
import type EditDnoteItemDialogEmits from './edit-dnote-item-dialog-emits';
import type EditDnoteItemDialogProps from './edit-dnote-item-dialog-props';
import DnoteItem from '@/classes/dnote/dnote-item';
import { useBackToCloseDialog } from '@/classes/use-back-to-close-dialog'
const is_show_dialog: Ref<boolean> = ref(false)
useBackToCloseDialog(is_show_dialog)

defineExpose({ show, hide })
defineProps<EditDnoteItemDialogProps>()
const model_value = defineModel<DnoteItem>()
const emits = defineEmits<EditDnoteItemDialogEmits>()

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
