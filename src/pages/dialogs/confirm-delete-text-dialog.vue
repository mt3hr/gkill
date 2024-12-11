<template>
    <v-dialog v-model="is_show_dialog">
        <ConfirmDeleteTextView :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="[text.generate_info_identifer()]" :kyou="kyou" :last_added_tag="last_added_tag" :text="text"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" 
            @requested_close_dialog="hide()"
            />
    </v-dialog>
</template>
<script setup lang="ts">
import { type Ref, ref, watch } from 'vue'
import type { ConfirmDeleteTextDialogProps } from './confirm-delete-text-dialog-props'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import ConfirmDeleteTextView from '../views/confirm-delete-text-view.vue'

const props = defineProps<ConfirmDeleteTextDialogProps>()
const emits = defineEmits<KyouDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
