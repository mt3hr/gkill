<template>
    <v-dialog v-model="is_show_dialog">
        <ShowPasswordResetLinkView :application_config="application_config" :gkill_api="gkill_api"
            :server_config="server_config" :user_id="user_id"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)" />
    </v-dialog>

</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import type { ShowPasswordResetLinkDialogEmits } from './show-password-reset-link-dialog-emits'
import type { ShowPasswordResetLinkDialogProps } from './show-password-reset-link-dialog-props'
import ShowPasswordResetLinkView from '../views/show-password-reset-link-view.vue'

const props = defineProps<ShowPasswordResetLinkDialogProps>()
const emits = defineEmits<ShowPasswordResetLinkDialogEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
