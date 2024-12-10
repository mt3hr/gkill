<template>
    <v-dialog v-model="is_show_dialog">
        <ConfirmDeleteKyouView :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="[kyou.generate_info_identifer()]" :kyou="kyou" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { type Ref, ref, watch } from 'vue'
import type { ConfirmDeleteKyouDialogProps } from './confirm-delete-kyou-dialog-props'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import ConfirmDeleteKyouView from '../views/confirm-delete-kyou-view.vue'
import type { ConfirmDeleteKyouDialogEmits } from './confirm-delete-kyou-dialog-emits';

const props = defineProps<ConfirmDeleteKyouDialogProps>()
const emits = defineEmits<ConfirmDeleteKyouDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
