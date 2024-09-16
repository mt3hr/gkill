<template>
    <v-dialog v-model="is_show_dialog">
        <StartTimeIsView :application_config="application_config" :gkill_api="gkill_api"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyou, is_checked) => emits('requested_update_check_kyous', kyou, is_checked)" />
    </v-dialog>
</template>
<script lang="ts" setup>
import { type Ref, ref } from 'vue';
import type { KyouViewEmits } from '../views/kyou-view-emits';
import type { StartTimeIsDialogProps } from './start-time-is-dialog-props';
import StartTimeIsView from '../views/start-time-is-view.vue';

const props = defineProps<StartTimeIsDialogProps>();
const emits = defineEmits<KyouViewEmits>();
defineExpose({ show, hide })

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
