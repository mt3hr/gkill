<template>
    <v-dialog v-model="is_show_dialog">
        <AddUrlogView :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')"
            @requested_update_check_kyous="(kyou, is_checked) => emits('requested_update_check_kyous', kyou, is_checked)"
            @requested_close_dialog="hide()" />
    </v-dialog>
</template>
<script setup lang="ts">
import { type Ref, ref, watch } from 'vue'
import type { EditLantanaDialogProps } from './edit-lantana-dialog-props'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import AddUrlogView from '../views/add-urlog-view.vue';

const props = defineProps<EditLantanaDialogProps>()
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
