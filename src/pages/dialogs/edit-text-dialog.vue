<template>
    <v-dialog :width="'fit-content'" v-model="is_show_dialog">
        <EditTextView :application_config="application_config" :gkill_api="gkill_api"
            :highlight_targets="text_highlight_targets" :kyou="kyou" :last_added_tag="last_added_tag" :text="text"
            :enable_context_menu="enable_context_menu" :enable_dialog="enable_dialog"
            @deleted_kyou="(deleted_kyou) => emits('deleted_kyou', deleted_kyou)"
            @deleted_tag="(deleted_tag) => emits('deleted_tag', deleted_tag)"
            @deleted_text="(deleted_text) => emits('deleted_text', deleted_text)"
            @deleted_notification="(deleted_notification) => emits('deleted_notification', deleted_notification)"
            @registered_kyou="(registered_kyou) => emits('registered_kyou', registered_kyou)"
            @registered_tag="(registered_tag) => emits('registered_tag', registered_tag)"
            @registered_text="(registered_text) => emits('registered_text', registered_text)"
            @registered_notification="(registered_notification) => emits('registered_notification', registered_notification)"
            @updated_kyou="(updated_kyou) => emits('updated_kyou', updated_kyou)"
            @updated_tag="(updated_tag) => emits('updated_tag', updated_tag)"
            @updated_text="(updated_text) => emits('updated_text', updated_text)"
            @updated_notification="(updated_notification) => emits('updated_notification', updated_notification)" 
            @received_errors="(errors) => emits('received_errors', errors)"
            @received_messages="(messages) => emits('received_messages', messages)"
            @requested_reload_kyou="(kyou) => emits('requested_reload_kyou', kyou)"
            @requested_reload_list="emits('requested_reload_list')" @requested_close_dialog="hide()"
            @requested_update_check_kyous="(kyou, is_checked) => emits('requested_update_check_kyous', kyou, is_checked)" />
    </v-dialog>
</template>
<script setup lang="ts">
import { computed, type Ref, ref } from 'vue'
import type { EditTextDialogProps } from './edit-text-dialog-props'
import type { KyouDialogEmits } from '../views/kyou-dialog-emits'
import EditTextView from '../views/edit-text-view.vue'
import type { InfoIdentifier } from '@/classes/datas/info-identifier';

const props = defineProps<EditTextDialogProps>()
const emits = defineEmits<KyouDialogEmits>()
defineExpose({ show, hide })

const text_highlight_targets = computed<Array<InfoIdentifier>>(() => {
    const info_identifer = props.text.generate_info_identifer()
    return [info_identifer]
})

const is_show_dialog: Ref<boolean> = ref(false)

async function show(): Promise<void> {
    is_show_dialog.value = true
}
async function hide(): Promise<void> {
    is_show_dialog.value = false
}
</script>
