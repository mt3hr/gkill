<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <EditIDFKyouView :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="[kyou.generate_info_identifer()]" :idf_kyou="kyou.typed_idf_kyou" :kyou="kyou"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog" :last_added_tag="last_added_tag"
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')" @requested_close_dialog="hide()"
            @requested_update_check_kyous="(kyou, is_checked) => emits('requested_update_check_kyous', kyou, is_checked)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { type Ref, ref } from 'vue'
import type { EditIDFKyouDialogProps } from './edit-idf-kyou-dialog-props'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import EditIDFKyouView from '../views/edit-idf-kyou-view.vue'

defineProps<EditIDFKyouDialogProps>()
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
