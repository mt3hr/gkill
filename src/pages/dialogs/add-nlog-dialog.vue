<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <AddNlogView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyou, is_checked) => emits('requested_update_check_kyous', kyou, is_checked)"
            @requested_close_dialog="hide()" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue'
import AddNlogView from '../views/add-nlog-view.vue'
import type { AddNlogDialogProps } from './add-nlog-dialog-props'
import type { KyouViewEmits } from '../views/kyou-view-emits'

defineProps<AddNlogDialogProps>()
const emits = defineEmits<KyouViewEmits>()
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
