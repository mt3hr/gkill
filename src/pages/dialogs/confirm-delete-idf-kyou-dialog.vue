<template>
    <v-dialog v-model="is_show_dialog">
        <ConfirmDeleteKyouView :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="[kyou.generate_info_identifer()]" :kyou="kyou" :last_added_tag="last_added_tag"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" @requested_close_dialog="hide()"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { type Ref, ref, watch } from 'vue'
import ConfirmDeleteKyouView from '../views/confirm-delete-kyou-view.vue';
import type { ConfirmDeleteIDFKyouDialogEmits } from './confirm-delete-idf-kyou-dialog-emits';
import type { ConfirmDeleteIDFKyouDialogProps } from './confirm-delete-idf-kyou-dialog-props';

const props = defineProps<ConfirmDeleteIDFKyouDialogProps>()
const emits = defineEmits<ConfirmDeleteIDFKyouDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
